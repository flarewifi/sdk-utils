//go:build !mono

// Non-mono system update orchestration (model A).
//
// In a non-mono build the core and each plugin are versioned and shipped as
// independent, already-built artifacts. A Go plugin .so is ABI-locked to the exact
// core build it loads into, so a core update cannot reuse the plugin .so files
// already on disk — every plugin must be REBUILT against the new core version and
// applied together with it, in one boot. That is "model A".
//
// StageSystemUpdate carries this out: it stages the self-contained core tarball
// plus every store plugin (rebuilt by the cloud against the TARGET core version)
// into the unified staging root (data/storage/system/updates/{pkg}), then writes
// the .staged_complete marker. The non-mono boot script (start.sh) overlays the
// whole set atomically on the next reboot. Progress is reported through the same
// download atomics the mono flow uses, so the existing download page/status
// controllers render it unchanged.
package updates

import (
	"context"
	"core/internal/api"
	"core/internal/plugindeps"
	"core/utils/config"
	"core/utils/plugins"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// Progress-bar bands (0..100) for the staged system update, in order:
//   - core tarball download:        0 .. coreDownloadEndPct
//   - store plugins (cloud build):  coreDownloadEndPct .. storePluginsEndPct
//   - local plugins (on-device rebuild against the staged core): storePluginsEndPct .. 99
const (
	coreDownloadEndPct = 60
	storePluginsEndPct = 80
)

// IsDownloaded reports whether an update has finished staging and is ready to
// apply on reboot. It is true once the .staged_complete marker exists (written
// last, so a mid-orchestration partial never qualifies). The marker covers BOTH
// kinds of staged set: a full system update (core + ABI-matched plugins) and a
// plugin-only update (latest plugins rebuilt against the unchanged current core).
// start.sh overlays whatever is staged atomically on the next boot, so both feed
// the same download → done → reboot flow. Use plugins.HasPendingCoreUpdate() where
// the distinction between a core and a plugin-only stage actually matters.
func IsDownloaded() bool {
	return sdkutils.FsExists(plugins.StagedCompleteMarkerPath())
}

// StageSystemUpdate orchestrates a non-mono model-A system update in the
// background. Safe to call once; concurrent calls are ignored while a stage is in
// flight. Progress and errors surface through the shared download atomics.
func StageSystemUpdate(g *api.CoreGlobals, update *SoftwareReleaseUpdate) {
	if downloading.Load() {
		return
	}

	downloading.Store(true)
	downloadPercent.Store(0)
	prevPercent.Store(0)
	downloadedBytes.Store(0)
	totalSizeBytes.Store(0)
	downloadError.Store(nil)
	pluginUpdateApplied.Store(false)
	awaitingConfirm.Store(false)
	cancelRequested.Store(false)
	lastStagedPlugins.Store([]StagedComponent{})
	lastSkippedPlugins.Store([]PluginBuildFailure{})
	// A core update begins by downloading the core tarball; stageSystemUpdate flips
	// to PhaseCompiling once it starts staging plugins.
	setPhase(PhaseDownloading)

	go func() {
		defer downloading.Store(false)

		if err := stageSystemUpdate(g, update); err != nil {
			// A clean, admin-initiated cancel at the build-failure gate is not an error:
			// discard the staged set and exit quietly so the page redirects to the
			// updates index (the cancel endpoint does the redirect) rather than showing
			// a scary download error.
			if errors.Is(err, ErrUpdateCancelled) {
				_ = sdkutils.FsEmptyDir(sdkutils.PathSystemUpdateDir)
				return
			}
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("stage system update: %v", err))
			downloadError.Store(&err)
			// Discard the partial set so a stale/incomplete staging is never left
			// behind. The marker is only written on success, so start.sh would
			// ignore it anyway, but clearing it reclaims space immediately.
			_ = sdkutils.FsEmptyDir(sdkutils.PathSystemUpdateDir)
			return
		}

		downloadPercent.Store(100)
	}()
}

// StagePluginsUpdate orchestrates a PLUGIN-ONLY update (no core change) in the
// background. The core is already current, so every installed store plugin keeps
// its ABI; we simply re-fetch the latest build of each (compiled by the cloud
// against the machine's CURRENT core) and stage it, then re-pin meta bundles and
// write the staged-complete marker so start.sh overlays them on the next reboot.
// Progress/errors surface through the same download atomics as the core flow, so
// the existing download page/status/done controllers render it unchanged. Safe to
// call once; concurrent calls are ignored while a stage is in flight.
func StagePluginsUpdate(g *api.CoreGlobals) {
	if downloading.Load() {
		return
	}

	downloading.Store(true)
	downloadPercent.Store(0)
	prevPercent.Store(0)
	downloadedBytes.Store(0)
	totalSizeBytes.Store(0)
	downloadError.Store(nil)
	pluginUpdateApplied.Store(false)
	awaitingConfirm.Store(false)
	cancelRequested.Store(false)
	lastStagedPlugins.Store([]StagedComponent{})
	lastSkippedPlugins.Store([]PluginBuildFailure{})
	// A plugin-only update downloads no core tarball — the whole run is staging
	// plugins (cloud builds + on-device recompiles), so report compiling throughout.
	setPhase(PhaseCompiling)

	go func() {
		defer downloading.Store(false)

		if err := stagePluginsUpdate(g); err != nil {
			// Admin cancelled at the build-failure gate — quiet exit, no error shown.
			if errors.Is(err, ErrUpdateCancelled) {
				_ = sdkutils.FsEmptyDir(sdkutils.PathSystemUpdateDir)
				return
			}
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("stage plugins update: %v", err))
			downloadError.Store(&err)
			_ = sdkutils.FsEmptyDir(sdkutils.PathSystemUpdateDir)
			return
		}

		downloadPercent.Store(100)
	}()
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// stagePluginsUpdate stages the latest build of every installed store plugin
// against the machine's current core (coreVersion == "") into the unified staging
// root and re-pins meta bundles. No core tarball is downloaded — this is the
// plugin-only counterpart to stageSystemUpdate.
//
// Two outcomes:
//   - At least one store plugin was staged → write the staged-complete marker so
//     start.sh overlays the new .so files on the next reboot (reboot-to-apply).
//   - Nothing to stage (e.g. the only update is a meta bundle whose members are
//     local plugins) → the RepinMetaRecordsToLatest below already applied the
//     bundle-version bump live, so no marker/reboot is needed; flag it APPLIED so
//     the download flow can finish without prompting for a reboot.
func stagePluginsUpdate(g *api.CoreGlobals) error {
	pluginPkgs, err := storePluginPackages()
	if err != nil {
		return fmt.Errorf("enumerate store plugins: %w", err)
	}

	// Start from a clean staging root so a previous partial/aborted attempt can't
	// leak into this one.
	if err := sdkutils.FsEmptyDir(sdkutils.PathSystemUpdateDir); err != nil {
		return err
	}

	// Stage every store plugin, rebuilt by the cloud against the CURRENT core (empty
	// coreVersion) so each .so stays ABI-matched to the core it already boots with. A
	// plugin whose server-side build fails does NOT abort the run — there is no core
	// change here, so every other plugin keeps its working .so and can still update.
	// Failures are collected and resolved at the confirmation gate below; stagedCount
	// tracks the successes so an all-failed run doesn't write a reboot marker for nothing.
	stagedCount := 0
	failedPlugins := []PluginBuildFailure{}
	for i, pkg := range pluginPkgs {
		// Checked between plugins, not mid-build: a build already in flight always
		// finishes, so this can only ever skip work that hasn't started yet.
		if cancelRequested.Load() {
			return ErrUpdateCancelled
		}

		// Every store plugin — meta-bundle members included — updates to its own
		// latest here; there is no bundle version pin. A member no longer covered by
		// its bundle is already disabled (skipped by storePluginPackages), so this
		// only stages members the machine is still entitled to.
		if err := g.PluginMgr.StagePluginUpdate(pkg, "", ""); err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("software update: store plugin %q failed to build: %v", pkg, err))
			failedPlugins = append(failedPlugins, PluginBuildFailure{Package: pkg, Reason: buildFailureReason(err)})
		} else {
			stagedCount++
			recordStagedPlugin(StagedComponent{Package: pkg, Name: storePluginDisplayName(g, pkg)})
		}
		downloadPercent.Store(int32((i + 1) * 99 / len(pluginPkgs)))
	}

	// Build-failure gate: nothing is committed until the admin decides. Cancel skips
	// the whole update (returns ErrUpdateCancelled); continue applies the rest and
	// records the skipped plugins. A no-failure run passes straight through.
	if err := confirmOrCancelOnFailures(g, failedPlugins); err != nil {
		return err
	}

	// Refresh meta-bundle records to their current membership. Best-effort: a lookup
	// failure must not abort an otherwise-staged update. Dropped members are not
	// uninstalled here — a member that lost bundle coverage is disabled at the next
	// boot by ValidateStorePlugins if it is no longer paid for.
	if err := g.PluginMgr.RepinMetaRecordsToLatest(); err != nil {
		g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("repin meta records: %v", err))
	}

	if stagedCount == 0 {
		// Nothing was actually staged for reboot — either there were no store plugins,
		// or every one was skipped (server-side build failed). The repin above is the
		// only change and is already live, so signal the no-reboot terminal state
		// instead of writing a marker that would prompt a pointless reboot.
		pluginUpdateApplied.Store(true)
		return nil
	}

	// Commit last: start.sh applies the staged plugins atomically the moment this
	// marker exists.
	if err := plugins.MarkStagedComplete(); err != nil {
		return fmt.Errorf("write staged-complete marker: %w", err)
	}

	return nil
}

func stageSystemUpdate(g *api.CoreGlobals, update *SoftwareReleaseUpdate) error {
	if update == nil || !update.HasUpdate {
		return errors.New("no system update to stage")
	}

	// Start from a clean staging root so a previous partial/aborted attempt can't
	// leak into this one.
	if err := sdkutils.FsEmptyDir(sdkutils.PathSystemUpdateDir); err != nil {
		return err
	}

	// 1. Stage the core: download the self-contained core tarball and extract it
	//    into updates/com.flarego.core (drives 0..coreDownloadEndPct).
	coreDest := plugins.GetPendingUpdatePath(plugins.CorePkg)
	if err := downloadAndExtractCore(update, coreDest); err != nil {
		return fmt.Errorf("stage core: %w", err)
	}
	if !plugins.HasPendingCoreUpdate() {
		return errors.New("staged core payload is missing bin/flare (corrupt download)")
	}

	// core/product.json ships INSIDE the downloaded tarball already, fully and
	// correctly stamped by the cloud build (go/builder's writeProductVersion) with
	// both this exact release's product version and its encrypted brand_id/
	// device_config -- CloneAndParseRelease refuses to build a release without it, so
	// there is nothing left for the device to stamp client-side. (The non-mono
	// download used to be a product-agnostic, cache-shared core_arch_bin with no
	// per-release product.json of its own -- devices downloading that artifact never
	// received a product.json at all, so core/product.json's version stayed stuck at
	// whatever it was before the FIRST non-mono update, forever. The server now
	// resolves the update to the full per-partner release tarball instead (same
	// upload category mono devices fetch) -- see FindLatestNonMonoSoftwareRelease.
	// See product.Version()/product.BrandId().)
	//
	// The full release tarball also carries plugins/installed/<pkg> -- this
	// release's CURATED plugin set (whatever the superuser picked), not what THIS
	// device is actually entitled to. Entitlement stays governed exclusively by the
	// per-device loop below (storePluginPackages/StagePluginUpdate), so the release's
	// bundled plugins/installed/ must never reach the device's app dir: dropping it
	// onto disk would make boot's plugin loader (which only checks structural
	// validity + disabled/blocked markers, not plugins.json membership) load a
	// plugin the device never purchased.
	if err := pruneStagedPluginInstallDir(coreDest); err != nil {
		return fmt.Errorf("prune release-curated plugins/installed from staged core: %w", err)
	}

	// The release tarball bundles plugin SOURCES under data/plugins/{local,devel} so the
	// device can recompile local plugins against the new core (start.sh relocates these
	// into the persistent data dir on apply). But a package can ship source yet be
	// REGISTERED as store on this device — store plugins are rebuilt server-side, so
	// their source must NOT be kept on-device. Drop every bundled source whose registered
	// Src is not "local" before the apply step copies what remains.
	if err := pruneNonLocalPluginSources(coreDest); err != nil {
		return fmt.Errorf("prune non-local plugin sources from staged core: %w", err)
	}

	// Normalize "devel" local plugins into "local" ones: on a device the source lives
	// under data/plugins/local/<pkg> (relocated by the updater/image), but a plugin may
	// still be registered with a data/plugins/devel/<pkg> LocalPath. Rewrite those to
	// data/plugins/local/ (and move the on-disk source if needed) and persist
	// plugins.json, so the on-device recompile below resolves the source correctly.
	if err := convertDevelPluginsToLocal(); err != nil {
		return fmt.Errorf("convert devel plugins to local: %w", err)
	}

	// The core tarball is down and verified; everything from here is plugin staging
	// (cloud builds + on-device recompiles). Switch the page label off "Downloading".
	setPhase(PhaseCompiling)

	// Resolve the TARGET core (ABI) version from the staged core's plugin.json — the
	// authoritative ABI identity every plugin .so must be compiled against. This is the
	// CORE version (core/plugin.json "version", e.g. 1.1.13), which is INDEPENDENT of
	// update.Version, the per-partner PRODUCT version (e.g. 1.1.12-beta.3): the two
	// diverge whenever a release bumps the core repo without bumping the product label.
	// Using the product version here asks the cloud to build plugins for the wrong (or
	// a non-existent/deleted) core, failing with "no server-side core build available".
	// The staged core was already extracted + verified above, so read its version
	// directly; a read failure means the payload is corrupt (treat as a hard error).
	stagedCore, err := sdkutils.GetPluginInfoFromPath(filepath.Join(coreDest, "core"))
	if err != nil {
		return fmt.Errorf("read staged core version: %w", err)
	}
	targetCore := stagedCore.Version

	// 2. Stage every store plugin, REBUILT by the cloud against the target core
	//    version so each .so is ABI-matched to the core it will boot with. We pass
	//    the latest plugin release (version == "") since the latest plugin and
	//    latest core are co-developed and known to compile together.
	pluginPkgs, err := storePluginPackages()
	if err != nil {
		return fmt.Errorf("enumerate store plugins: %w", err)
	}

	// A store plugin failing its server-side build must NOT abort the whole update —
	// the core is already staged and the remaining plugins can still ship. Collect the
	// failures (both store here and local below) and let the admin decide at the gate
	// after both loops. Only com.flarego.core failing (handled by downloadAndExtractCore
	// above) is fatal.
	failedPlugins := []PluginBuildFailure{}
	for i, pkg := range pluginPkgs {
		// Checked between plugins, not mid-build (see stagePluginsUpdate). Cancelling
		// here discards the already-staged core too — the caller empties the whole
		// staging root on ErrUpdateCancelled — since nothing has committed yet.
		if cancelRequested.Load() {
			return ErrUpdateCancelled
		}

		if err := g.PluginMgr.StagePluginUpdate(pkg, "", targetCore); err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("software update: store plugin %q failed to build: %v", pkg, err))
			failedPlugins = append(failedPlugins, PluginBuildFailure{Package: pkg, Reason: buildFailureReason(err)})
		} else {
			recordStagedPlugin(StagedComponent{Package: pkg, Name: storePluginDisplayName(g, pkg)})
		}
		// Advance the bar across the store-plugin band.
		pct := coreDownloadEndPct + (i+1)*(storePluginsEndPct-coreDownloadEndPct)/len(pluginPkgs)
		downloadPercent.Store(int32(pct))
	}

	// 3. Stage every LOCAL plugin, recompiled ON-DEVICE against the STAGED core so
	//    each .so is ABI-matched to the core it will boot with. Local plugins have no
	//    cloud build; their source ships under data/plugins/local/ (persisted across
	//    overlays) and is recompiled here — during staging, before the reboot — so the
	//    progress bar reflects it. start.sh applies the staged package on reboot,
	//    atomically with the core, and can roll it back. Build against coreDest (the
	//    staged new core), NOT the live one.
	allLocalTargets, err := plugins.InstalledLocalPluginSrcDirs()
	if err != nil {
		return fmt.Errorf("enumerate local plugins: %w", err)
	}
	// Drop disabled/blocked local plugins up front (before computing the band
	// percentages below) so the boot loader's skip set and the recompile set agree —
	// a local plugin the loader won't load is not worth compiling against the new core.
	localTargets := filterStageableLocalPlugins(allLocalTargets)
	// Fetch the TARGET core version's dependency lock ONCE so every local plugin is
	// rebuilt against the same module versions+hashes as the staged core and the
	// cloud-built store plugins beside it. Empty/unreachable => nil => unpinned.
	pinnedDeps := plugindeps.Fetch(targetCore)
	for i, srcDir := range localTargets {
		// Checked between plugins, not mid-build (see the store-plugin loop above).
		if cancelRequested.Load() {
			return ErrUpdateCancelled
		}

		// Name the plugin currently compiling so the software-update logs trace
		// on-device recompile progress (store plugins are built in the cloud; these
		// local ones are the only ones that actually compile here). Fall back to the
		// source dir name if plugin.json can't be read.
		pluginName := filepath.Base(srcDir)
		pluginDisplayName := pluginName
		if info, err := sdkutils.GetPluginInfoFromPath(srcDir); err == nil {
			pluginName = info.Package
			pluginDisplayName = info.Name
		}
		g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("Compiling plugin %s (%d/%d)", pluginName, i+1, len(localTargets)))

		// A local plugin whose on-device recompile fails is collected (not fatal) and
		// resolved at the gate below — same policy as the store plugins above. Its old
		// .so stays on disk ABI-mismatched with the staged core, so the boot loader's
		// load-failure path handles it; only com.flarego.core failing is fatal.
		if err := plugins.StageLocalPluginRebuild(srcDir, coreDest, pinnedDeps); err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("software update: local plugin %q failed to build: %v", pluginName, err))
			failedPlugins = append(failedPlugins, PluginBuildFailure{Package: pluginName, Reason: buildFailureReason(err)})
		} else {
			recordStagedPlugin(StagedComponent{Package: pluginName, Name: pluginDisplayName})
		}
		pct := storePluginsEndPct + (i+1)*(99-storePluginsEndPct)/len(localTargets)
		downloadPercent.Store(int32(pct))
	}

	// Build-failure gate: with the core already staged, pause for the admin before
	// committing. Cancel discards the whole staged set (no update — not even the core);
	// continue applies the core plus the plugins that DID build and records the rest as
	// skipped. A no-failure run passes straight through.
	if err := confirmOrCancelOnFailures(g, failedPlugins); err != nil {
		return err
	}

	// The operator chose to continue past the build failures. Each skipped plugin's
	// old .so is ABI-locked to the PREVIOUS core and would fail to load against the
	// staged one; outside production that load — or the boot-time recompile of a local
	// plugin — aborts boot and start.sh rolls the WHOLE update back (the "reverts on
	// reboot" symptom). Flag each skipped plugin so the boot loader leaves its stale
	// .so alone: the new core then boots cleanly with the plugin absent until a later
	// update rebuilds it (which clears the marker by replacing the install dir).
	// Best-effort — a marker write must not abort an otherwise-committed update. Only
	// the CORE path does this; a plugin-only update keeps the current core, so a failed
	// plugin's existing .so is still ABI-valid and must keep loading.
	for _, f := range failedPlugins {
		if err := plugins.MarkUpdateSkipped(f.Package); err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("software update: failed to flag skipped plugin %q: %v", f.Package, err))
		}
	}

	// Refresh meta-bundle records to their current membership (only now that we're
	// committing). Members are ordinary store plugins already staged above; this only
	// advances the bundle metadata. Best-effort: a lookup failure must not abort the
	// update. Dropped members are not uninstalled — boot ValidateStorePlugins disables
	// any that are no longer paid for.
	if err := g.PluginMgr.RepinMetaRecordsToLatest(); err != nil {
		g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("repin meta records: %v", err))
	}

	// 4. Commit: the marker gates the boot-time apply. Written LAST so a crash
	//    mid-staging leaves an incomplete set that start.sh discards.
	if err := plugins.MarkStagedComplete(); err != nil {
		return fmt.Errorf("write staged-complete marker: %w", err)
	}

	return nil
}

// pruneStagedPluginInstallDir removes coreDest/plugins/installed entirely — the full
// release tarball's CURATED plugin set (see stageSystemUpdate) — so start.sh's blanket
// core-package overlay (cp -a onto $APP_DIR) can never install a plugin this specific
// device isn't entitled to. Per-device plugin entitlement is decided exclusively by the
// storePluginPackages/StagePluginUpdate loop later in stageSystemUpdate, which stages
// each entitled plugin as its OWN package under the staging root, applied by start.sh
// as a separate overlay onto plugins/installed/<pkg> — unaffected by this prune, since
// it targets the core package's copy of plugins/installed, not the per-plugin one.
func pruneStagedPluginInstallDir(coreDest string) error {
	installDir := filepath.Join(coreDest, "plugins", "installed")
	if err := os.RemoveAll(installDir); err != nil {
		return fmt.Errorf("remove staged plugins/installed: %w", err)
	}
	return nil
}

// pruneNonLocalPluginSources removes bundled plugin source trees from the staged core
// payload (coreDest/data/plugins/{local,devel}/<pkg>) for every package that is NOT
// registered as Src=local in the device's live data/config/plugins.json. The non-mono
// release bundles local plugin sources so the device can recompile them against the new
// core, but a package can ship source while being registered store (store-managed, built
// server-side) — keeping that source on-device would wrongly route it through the local
// recompile path. Only sources for genuinely local-registered plugins are kept; the rest
// (store, git, system, or unregistered) are dropped here, before start.sh's apply step
// relocates the survivors into the persistent data dir. Routing is by the registered
// metadata Src, not by where the source happens to sit.
func pruneNonLocalPluginSources(coreDest string) error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}

	localPkgs := make(map[string]struct{})
	for _, meta := range cfg.Metadata {
		if meta.Def.Src == sdkutils.PluginSrcLocal {
			localPkgs[meta.Package] = struct{}{}
		}
	}

	for _, sub := range []string{"local", "devel"} {
		srcRoot := filepath.Join(coreDest, "data", "plugins", sub)
		entries, err := os.ReadDir(srcRoot)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if _, ok := localPkgs[e.Name()]; ok {
				continue // registered local — keep its source for on-device recompile
			}
			if err := os.RemoveAll(filepath.Join(srcRoot, e.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

// convertDevelPluginsToLocal normalizes "devel" local plugins into "local" ones and
// persists the change to plugins.json. There is no "devel" plugin Src — a devel plugin is
// just a local plugin (Src=local) whose registered LocalPath points under
// data/plugins/devel/<pkg>, the build/dev source layout. On a device the non-mono release
// ships those sources under data/plugins/local/<pkg> (the updater and OS-image builder
// relocate them there), so a LocalPath still referencing devel/ can never be resolved by
// the on-device recompile. For every local-registered plugin whose LocalPath references
// data/plugins/devel/, rewrite it to data/plugins/local/ — moving the on-disk source dir
// too if it still sits under devel/ — and write the updated config back.
func convertDevelPluginsToLocal() error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}

	const develPrefix = "data/plugins/devel/"
	const localPrefix = "data/plugins/local/"

	changed := false
	for i := range cfg.Metadata {
		def := &cfg.Metadata[i].Def
		if def.Src != sdkutils.PluginSrcLocal {
			continue
		}

		lp := filepath.ToSlash(def.LocalPath)
		idx := strings.Index(lp, develPrefix)
		if idx < 0 {
			continue
		}

		pkgRel := lp[idx+len(develPrefix):] // <pkg>[/...]
		newLocalPath := localPrefix + pkgRel

		// Move the on-disk source from devel/ to local/ if it still lives there and the
		// destination is not already populated (the device usually already has it under
		// local/, in which case this is a no-op and only the config is rewritten).
		oldAbs := filepath.Join(sdkutils.PathAppDir, develPrefix+pkgRel)
		newAbs := filepath.Join(sdkutils.PathAppDir, newLocalPath)
		if sdkutils.FsExists(oldAbs) && !sdkutils.FsExists(newAbs) {
			if err := sdkutils.FsEnsureDir(filepath.Dir(newAbs)); err != nil {
				return err
			}
			if err := sdkutils.FsMoveDir(oldAbs, newAbs); err != nil {
				return err
			}
		}

		def.LocalPath = newLocalPath
		changed = true
	}

	if changed {
		if err := config.WritePluginsConfig(cfg); err != nil {
			return err
		}
	}

	return nil
}

// downloadAndExtractCore streams the self-contained core tarball to a temp file
// (verifying the MD5 checksum), reporting byte/percent progress through the shared
// download atomics, then extracts it into dest. The core tarball archives its
// contents at the tar root, so dest ends up holding bin/flare, core/, sdk/, etc.
func downloadAndExtractCore(update *SoftwareReleaseUpdate, dest string) error {
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := sdkutils.FsEnsureDir(dest); err != nil {
		return err
	}

	tmpFile := filepath.Join(sdkutils.PathTmpDir, "core-update-"+sdkutils.RandomStr(8)+".tar.gz")
	if err := sdkutils.FsEnsureDir(filepath.Dir(tmpFile)); err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	resp, err := http.Get(update.ReleseFileURL)
	if err != nil {
		return ErrDownload
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ErrDownload
	}

	totalSize := resp.ContentLength
	if totalSize <= 0 {
		return ErrDownload
	}
	totalSizeBytes.Store(totalSize)

	file, err := os.Create(tmpFile)
	if err != nil {
		return ErrDownload
	}
	defer file.Close()

	hasher := md5.New()
	writer := io.MultiWriter(file, hasher)
	downloaded := int64(0)
	buf := make([]byte, 32*1024)

	for {
		if cancelRequested.Load() {
			return ErrUpdateCancelled
		}

		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := writer.Write(buf[:n]); werr != nil {
				return ErrDownload
			}
			downloaded += int64(n)
			downloadedBytes.Store(downloaded)
			downloadPercent.Store(int32(downloaded * coreDownloadEndPct / totalSize))
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return ErrDownload
		}
	}

	if update.ReleaseFileChecksum != "" {
		actual := base64.StdEncoding.EncodeToString(hasher.Sum(nil))
		if actual != update.ReleaseFileChecksum {
			return ErrChecksumMismatch
		}
	}

	if err := sdkutils.Untar(tmpFile, dest); err != nil {
		return ErrExtract
	}

	return nil
}

// storePluginPackages returns the package ids of every store-sourced plugin
// installed on this machine, excluding meta bundles (which have no .so of their
// own). These are the plugins that must be rebuilt against the new core version.
func storePluginPackages() ([]string, error) {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return nil, err
	}

	metaPkgs := make(map[string]struct{}, len(cfg.MetaPlugins))
	for _, m := range cfg.MetaPlugins {
		metaPkgs[m.Package] = struct{}{}
	}

	pkgs := []string{}
	for _, meta := range plugins.InstalledPluginsList() {
		if meta.Def.Src != sdkutils.PluginSrcStore {
			continue
		}
		if _, isMeta := metaPkgs[meta.Package]; isMeta {
			continue
		}
		// Don't rebuild a plugin the boot loader will skip anyway (see
		// skipStagingForUpdate). For a store plugin this also avoids a hard failure:
		// the cloud withholds the download URL for an unpaid plugin, which would
		// otherwise abort the whole staged update at "build completed without a
		// download url".
		if skipStagingForUpdate(meta.Package) {
			continue
		}
		pkgs = append(pkgs, meta.Package)
	}

	return pkgs, nil
}

// skipStagingForUpdate reports whether a plugin must be excluded from the staged
// software update — it is DISABLED or BLOCKED, the same two markers the boot
// loader honors (see boot.InitPlugins), so the freshly built .so would never be
// loaded. "Disabled" is also how a lapsed purchase is represented on-device:
// boot.ValidateStorePlugins writes the disabled marker for any store plugin whose
// payment is required but not satisfied. So a disabled OR payment-lapsed plugin is
// skipped here, and "blocked" (cloud denylist) is skipped for the same reason —
// compiling something that won't boot is wasted work (and, for store plugins,
// would fail the whole update).
func skipStagingForUpdate(pkg string) bool {
	return plugins.IsDisabled(pkg) || plugins.IsBlocked(pkg)
}

// filterStageableLocalPlugins drops local plugin source dirs whose plugin is
// disabled or blocked (skipStagingForUpdate), so the on-device recompile step
// matches the boot loader's skip set. A dir whose plugin.json can't be read is
// kept (its package — and thus its marker state — is unknown; recompiling and
// letting the loader decide is the safe fallback, mirroring the loop below).
func filterStageableLocalPlugins(srcDirs []string) []string {
	stageable := make([]string, 0, len(srcDirs))
	for _, srcDir := range srcDirs {
		info, err := sdkutils.GetPluginInfoFromPath(srcDir)
		if err == nil && skipStagingForUpdate(info.Package) {
			continue
		}
		stageable = append(stageable, srcDir)
	}
	return stageable
}

// confirmOrCancelOnFailures gates the commit step on the admin's decision when one or
// more plugins failed to build. With no failures it returns nil immediately (proceed).
// Otherwise it PAUSES staging until the admin resolves the dialog and either:
//   - cancel → returns ErrUpdateCancelled so the caller discards the whole staged set
//     (no update applies, not even the core); or
//   - continue → records each skipped plugin for the admin's audit trail and returns
//     nil so the caller commits the core plus the plugins that did build.
//
// The detailed per-plugin build error was already logged in the staging loops; the
// audit notification below stays generic (names only the package), per the
// error-message hygiene rules.
func confirmOrCancelOnFailures(g *api.CoreGlobals, failed []PluginBuildFailure) error {
	if len(failed) == 0 {
		return nil
	}

	if !waitForPluginFailureDecision(failed) {
		return ErrUpdateCancelled
	}

	// Persist the final skipped set for the download-done page's summary — the
	// dialog's own copy (confirmFailed) is cleared by waitForPluginFailureDecision
	// once the admin's choice is read, so this is the only record left afterward.
	lastSkippedPlugins.Store(append([]PluginBuildFailure(nil), failed...))

	for _, f := range failed {
		notifyPluginUpdateSkipped(g, f.Package)
	}
	return nil
}

// storePluginDisplayName resolves a store plugin's human-readable name for the
// download-done summary, falling back to the package id if it isn't currently
// loaded (e.g. a first-time install staged alongside a core update).
func storePluginDisplayName(g *api.CoreGlobals, pkg string) string {
	if p, ok := g.PluginMgr.FindByPkg(pkg); ok {
		return p.Info().Name
	}
	return pkg
}

// buildFailureReason extracts a clean, user-facing reason from a staging error: the
// cloud's PluginBuildError message when present (a disabled plugin, or a compile
// error reported by the server build), otherwise the error text (e.g. a local
// plugin's on-device recompile failure).
func buildFailureReason(err error) string {
	var be *api.PluginBuildError
	if errors.As(err, &be) && be.Reason != "" {
		return be.Reason
	}
	return err.Error()
}

// notifyPluginUpdateSkipped raises an admin notification that a plugin was skipped
// during the software update (its build failed and the admin chose to continue with
// the rest). It is the persistent record behind the transient confirmation dialog.
// Always notifies (not gated on env) — the operator explicitly triggered the update
// and must see which plugins it left behind.
func notifyPluginUpdateSkipped(g *api.CoreGlobals, pkg string) {
	subject := g.CoreAPI.Translate("error", "Plugin update skipped")
	content := g.CoreAPI.Translate("error", "The plugin <% .pkg %> could not be built and was skipped during the software update. The rest of the update was applied; try updating this plugin again later", "pkg", pkg)

	if err := g.CoreAPI.Notification().AddNotification(context.Background(), sdkapi.AddNotificationParams{
		Subject: subject,
		Content: content,
		Type:    sdkapi.NotificationTypeError,
	}); err != nil {
		g.CoreAPI.Logger().Error(fmt.Sprintf("failed to notify admin that plugin %q was skipped during update: %v", pkg, err))
	}
}
