//go:build !mono

// Plugin update discovery for non-mono builds.
//
// In a non-mono build the core binary and each plugin are versioned and shipped
// independently (plugins are separate plugin.so files under plugins/installed).
// This file enumerates the store-sourced plugins installed on the machine and
// asks the cloud for the latest release of each, so the Software Updates page can
// list per-plugin updates alongside the core/system update.
//
// Only store-sourced plugins (Def.Src == store) are checked — they are the only
// source with a version-lookup RPC (FetchLatestPluginReleaseByPackage).
//
// Meta bundles are surfaced as a SINGLE row each (keyed by the bundle package),
// not as their individual members. A bundle's installed version comes from its
// meta record (cfg.MetaPlugins) and its latest version from the cloud; its owned
// members are excluded from the per-plugin list so the user manages the bundle as
// one unit. Members the user ALSO installed standalone still appear on their own.
package updates

import (
	"core/internal/api"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v3"
	"core/utils/config"
	"core/utils/plugins"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	sdkutils "github.com/flarewifi/sdk-utils"
)

// pendingUpdate carries everything needed to build a PluginUpdate row once the
// batched cloud response comes back. The installed version + display name are
// captured up front (locally) so the response only has to supply the LATEST
// version; isMeta selects which branch of the response to read.
type pendingUpdate struct {
	pkg     string
	name    string
	current *semver.Version
	isMeta  bool
}

// CheckPluginUpdates returns the update status of every store-sourced plugin
// installed on this machine. Each plugin's installed version is read from its
// loaded plugin.json and compared against the latest release reported by the
// cloud. Plugins whose release lookup fails are skipped (best-effort) rather than
// failing the whole check. Meta bundles are returned as a single row each (keyed
// by the bundle package); their owned members are not listed individually.
//
// All packages are resolved in a SINGLE batched RPC
// (FetchLatestPluginReleasesByPackages) rather than one request per plugin. A
// machine installs dozens of store plugins, and the old per-plugin loop issued
// one HTTP request each — so when the machine was offline or cloud-sync was
// disabled the admin log filled with a "failed to fetch the latest release for
// <pkg>" line per plugin. Batching collapses a whole-check transport failure into
// a single log line and any per-package misses into one aggregated line.
func CheckPluginUpdates(g *api.CoreGlobals) ([]PluginUpdate, error) {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return nil, err
	}

	// Packages that are meta bundles — surfaced as their own bundle rows below and
	// excluded from the per-plugin loop.
	metaPkgs := make(map[string]struct{}, len(cfg.MetaPlugins))
	for _, m := range cfg.MetaPlugins {
		metaPkgs[m.Package] = struct{}{}
	}
	// Members owned by a bundle (and not also installed standalone) are managed
	// through their bundle, so they are hidden from the per-plugin list.
	hiddenMembers := metaOwnedMembers(cfg)

	// Phase 1 (local): gather every package to check along with the local metadata
	// needed to build its row. Bad LOCAL versions are logged here (rare, genuinely
	// diagnostic) and dropped — they never reach the batched request.
	var pendings []pendingUpdate

	// One row per meta bundle: installed version from the meta record.
	for _, m := range cfg.MetaPlugins {
		current, err := semver.NewVersion(m.Version)
		if err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("plugin update check: bad current version %q for meta %s: %v", m.Version, m.Package, err))
			continue
		}
		pendings = append(pendings, pendingUpdate{pkg: m.Package, name: m.Name, current: current, isMeta: true})
	}

	for _, meta := range plugins.InstalledPluginsList() {
		// Only store-sourced plugins can be version-checked via the store RPC.
		if meta.Def.Src != sdkutils.PluginSrcStore {
			continue
		}
		if _, isMeta := metaPkgs[meta.Package]; isMeta {
			continue
		}
		if _, hidden := hiddenMembers[meta.Package]; hidden {
			continue
		}

		// Installed version + display name come from the loaded plugin.
		p, ok := g.PluginMgr.FindByPkg(meta.Package)
		if !ok {
			continue
		}
		info := p.Info()

		current, err := semver.NewVersion(info.Version)
		if err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("plugin update check: bad current version %q for %s: %v", info.Version, meta.Package, err))
			continue
		}
		pendings = append(pendings, pendingUpdate{pkg: meta.Package, name: info.Name, current: current, isMeta: false})
	}

	updateList := []PluginUpdate{}
	if len(pendings) == 0 {
		return updateList, nil
	}

	// Phase 2 (one request): resolve every package's latest release in a single
	// batched call.
	reqs := make([]*rpc_flarewifi_v3.FetchLatestPluginReleaseByPackageRequest, 0, len(pendings))
	for _, p := range pendings {
		reqs = append(reqs, &rpc_flarewifi_v3.FetchLatestPluginReleaseByPackageRequest{PluginPackage: p.pkg})
	}

	srv, ctx := rpc.GetTwirpServiceAndCtx()
	batch, err := srv.FetchLatestPluginReleasesByPackages(ctx, &rpc_flarewifi_v3.FetchLatestPluginReleasesByPackagesRequest{
		Requests: reqs,
	})
	if err != nil {
		// A single transport failure for the WHOLE check (offline / cloud-sync
		// disabled / not activated). Log ONCE — not once per plugin — and abort.
		// Log only the method; the raw RPC error exposes the cloud endpoint URL.
		g.CoreAPI.LoggerAPI.Error("plugin update check: failed to fetch the latest plugin releases")
		return nil, err
	}

	// Index results by package so rows map back order-independently.
	byPkg := make(map[string]*rpc_flarewifi_v3.PluginReleaseResult, len(batch.GetResults()))
	for _, res := range batch.GetResults() {
		byPkg[res.GetPluginPackage()] = res
	}

	// Phase 3 (local): build the update rows from the batched results.
	var failedPkgs []string
	for _, p := range pendings {
		res, ok := byPkg[p.pkg]
		if !ok || res.GetError() != "" || res.GetResponse() == nil {
			// This package's lookup failed on the cloud (not found / off-channel /
			// gated). Collected for a single aggregated log line below.
			failedPkgs = append(failedPkgs, p.pkg)
			continue
		}
		resp := res.GetResponse()

		if p.isMeta {
			latest, err := semver.NewVersion(resp.GetVersion())
			if err != nil {
				continue
			}
			updateList = append(updateList, PluginUpdate{
				Package:        p.pkg,
				Name:           p.name,
				CurrentVersion: p.current.String(),
				LatestVersion:  latest.String(),
				HasUpdate:      latest.GreaterThan(p.current),
				IsMeta:         true,
			})
			continue
		}

		// Skip bundles surfaced as meta by the cloud — nothing to download here.
		if resp.GetIsMeta() {
			continue
		}
		rel := resp.GetPluginRelease()
		if rel == nil {
			continue
		}
		latestStr := fmt.Sprintf("%d.%d.%d", rel.GetMajor(), rel.GetMinor(), rel.GetPatch())
		latest, err := semver.NewVersion(latestStr)
		if err != nil {
			continue
		}

		updateList = append(updateList, PluginUpdate{
			Package:        p.pkg,
			Name:           p.name,
			CurrentVersion: p.current.String(),
			LatestVersion:  latest.String(),
			HasUpdate:      latest.GreaterThan(p.current),
		})
	}

	// Per-package misses (the cloud was reachable but some packages had no
	// resolvable release) are collapsed into ONE line instead of one per plugin.
	if len(failedPkgs) > 0 {
		g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("plugin update check: failed to fetch the latest release for %d plugin(s): %s", len(failedPkgs), strings.Join(failedPkgs, ", ")))
	}

	return updateList, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// metaOwnedMembers returns the set of plugin packages that belong to a meta
// bundle and are NOT also installed standalone. These are managed through their
// bundle, so they are excluded from the per-plugin update list. A member the user
// installed on its own (Standalone metadata flag) is omitted from the set so it
// still appears as its own row.
func metaOwnedMembers(cfg sdkutils.PluginsConfig) map[string]struct{} {
	standalone := make(map[string]struct{}, len(cfg.Metadata))
	for _, md := range cfg.Metadata {
		if md.Standalone {
			standalone[md.Package] = struct{}{}
		}
	}

	owned := map[string]struct{}{}
	for _, m := range cfg.MetaPlugins {
		for _, member := range m.Members {
			if _, ok := standalone[member]; ok {
				continue
			}
			owned[member] = struct{}{}
		}
	}
	return owned
}
