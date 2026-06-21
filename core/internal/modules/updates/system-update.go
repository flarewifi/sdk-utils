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
	// A core update begins by downloading the core tarball; stageSystemUpdate flips
	// to PhaseCompiling once it starts staging plugins.
	setPhase(PhaseDownloading)

	go func() {
		defer downloading.Store(false)

		if err := stageSystemUpdate(g, update); err != nil {
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
	// A plugin-only update downloads no core tarball — the whole run is staging
	// plugins (cloud builds + on-device recompiles), so report compiling throughout.
	setPhase(PhaseCompiling)

	go func() {
		defer downloading.Store(false)

		if err := stagePluginsUpdate(g); err != nil {
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
	// coreVersion) so each .so stays ABI-matched to the core it already boots with.
	for i, pkg := range pluginPkgs {
		if err := g.PluginMgr.StagePluginUpdate(pkg, "", ""); err != nil {
			return fmt.Errorf("stage plugin %s: %w", pkg, err)
		}
		downloadPercent.Store(int32((i + 1) * 99 / len(pluginPkgs)))
	}

	// Re-pin meta-bundle records to their latest version so a bundle stops showing
	// "update available" once its members are refreshed (or, for local-member
	// bundles, applies the version bump outright). Best-effort: a lookup failure
	// must not abort an otherwise-staged update.
	if err := g.PluginMgr.RepinMetaRecordsToLatest(); err != nil {
		g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("repin meta records: %v", err))
	}

	if len(pluginPkgs) == 0 {
		// Nothing was staged for reboot — the repin above is the whole change and is
		// already live. Signal the no-reboot terminal state instead of a marker.
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

	// 2. Stage every store plugin, REBUILT by the cloud against the target core
	//    version so each .so is ABI-matched to the core it will boot with. We pass
	//    the latest plugin release (version == "") since the latest plugin and
	//    latest core are co-developed and known to compile together.
	targetCore := update.Version.String()
	pluginPkgs, err := storePluginPackages()
	if err != nil {
		return fmt.Errorf("enumerate store plugins: %w", err)
	}

	for i, pkg := range pluginPkgs {
		if err := g.PluginMgr.StagePluginUpdate(pkg, "", targetCore); err != nil {
			return fmt.Errorf("stage plugin %s: %w", pkg, err)
		}
		// Advance the bar across the store-plugin band.
		pct := coreDownloadEndPct + (i+1)*(storePluginsEndPct-coreDownloadEndPct)/len(pluginPkgs)
		downloadPercent.Store(int32(pct))
	}

	// Re-pin meta-bundle records to their latest version. Members are ordinary
	// store plugins already staged above; this only advances the bundle metadata
	// so a bundle stops showing "update available" after its members are refreshed.
	// Best-effort: a lookup failure must not abort an otherwise-staged update.
	if err := g.PluginMgr.RepinMetaRecordsToLatest(); err != nil {
		g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("repin meta records: %v", err))
	}

	// 3. Stage every LOCAL plugin, recompiled ON-DEVICE against the STAGED core so
	//    each .so is ABI-matched to the core it will boot with. Local plugins have no
	//    cloud build; their source ships under data/plugins/local/ (persisted across
	//    overlays) and is recompiled here — during staging, before the reboot — so the
	//    progress bar reflects it. start.sh applies the staged package on reboot,
	//    atomically with the core, and can roll it back. Build against coreDest (the
	//    staged new core), NOT the live one.
	localTargets, err := plugins.InstalledLocalPluginSrcDirs()
	if err != nil {
		return fmt.Errorf("enumerate local plugins: %w", err)
	}
	// Fetch the TARGET core version's dependency lock ONCE so every local plugin is
	// rebuilt against the same module versions+hashes as the staged core and the
	// cloud-built store plugins beside it. Empty/unreachable => nil => unpinned.
	pinnedDeps := plugindeps.Fetch(targetCore)
	for i, srcDir := range localTargets {
		// Name the plugin currently compiling so the software-update logs trace
		// on-device recompile progress (store plugins are built in the cloud; these
		// local ones are the only ones that actually compile here). Fall back to the
		// source dir name if plugin.json can't be read.
		pluginName := filepath.Base(srcDir)
		if info, err := sdkutils.GetPluginInfoFromPath(srcDir); err == nil {
			pluginName = info.Package
		}
		g.CoreAPI.LoggerAPI.Info(fmt.Sprintf("Compiling plugin %s (%d/%d)", pluginName, i+1, len(localTargets)))

		if err := plugins.StageLocalPluginRebuild(srcDir, coreDest, pinnedDeps); err != nil {
			return fmt.Errorf("stage local plugin %s: %w", srcDir, err)
		}
		pct := storePluginsEndPct + (i+1)*(99-storePluginsEndPct)/len(localTargets)
		downloadPercent.Store(int32(pct))
	}

	// 4. Commit: the marker gates the boot-time apply. Written LAST so a crash
	//    mid-staging leaves an incomplete set that start.sh discards.
	if err := plugins.MarkStagedComplete(); err != nil {
		return fmt.Errorf("write staged-complete marker: %w", err)
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
		pkgs = append(pkgs, meta.Package)
	}

	return pkgs, nil
}
