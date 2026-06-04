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
	"core/internal/rpc/rpc_flarewifi_v2"
	"core/utils/config"
	"core/utils/plugins"
	"fmt"

	"github.com/Masterminds/semver/v3"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

// CheckPluginUpdates returns the update status of every store-sourced plugin
// installed on this machine. Each plugin's installed version is read from its
// loaded plugin.json and compared against the latest release reported by the
// cloud. Plugins whose release lookup fails are skipped (best-effort) rather than
// failing the whole check. Meta bundles are returned as a single row each (keyed
// by the bundle package); their owned members are not listed individually.
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

	srv, ctx := rpc.GetTwirpServiceAndCtx()

	updateList := []PluginUpdate{}

	// One row per meta bundle: installed version from the meta record, latest from
	// the cloud. A failed lookup skips that bundle (best-effort) rather than failing
	// the whole check.
	for _, m := range cfg.MetaPlugins {
		current, err := semver.NewVersion(m.Version)
		if err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("plugin update check: bad current version %q for meta %s: %v", m.Version, m.Package, err))
			continue
		}

		resp, err := srv.FetchLatestPluginReleaseByPackage(ctx, &rpc_flarewifi_v2.FetchLatestPluginReleaseByPackageRequest{
			PluginPackage: m.Package,
		})
		if err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("plugin update check: fetch release for meta %s: %v", m.Package, err))
			continue
		}

		latest, err := semver.NewVersion(resp.GetVersion())
		if err != nil {
			continue
		}

		updateList = append(updateList, PluginUpdate{
			Package:        m.Package,
			Name:           m.Name,
			CurrentVersion: current.String(),
			LatestVersion:  latest.String(),
			HasUpdate:      latest.GreaterThan(current),
			IsMeta:         true,
		})
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

		resp, err := srv.FetchLatestPluginReleaseByPackage(ctx, &rpc_flarewifi_v2.FetchLatestPluginReleaseByPackageRequest{
			PluginPackage: meta.Package,
		})
		if err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("plugin update check: fetch release for %s: %v", meta.Package, err))
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
			Package:        meta.Package,
			Name:           info.Name,
			CurrentVersion: current.String(),
			LatestVersion:  latest.String(),
			HasUpdate:      latest.GreaterThan(current),
		})
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
