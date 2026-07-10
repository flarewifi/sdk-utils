//go:build !mono

package plugins

import (
	"fmt"
	"os"
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// StageLocalPluginRebuild recompiles a local plugin's plugin.so against the STAGED
// (not-yet-applied) new core and assembles an install-ready package under the unified
// staging dir (data/storage/system/updates/<pkg>) so start.sh applies it on the next
// reboot — atomically with the core it was built against.
//
// This is the non-mono "model A" path for LOCAL plugins. A Go plugin.so is ABI-locked
// to the exact core build it loads into; store plugins are rebuilt server-side and
// staged, but local plugins have no cloud build, so we recompile their on-device
// source (shipped under data/plugins/local/<pkg>) here. We do it during STAGING —
// before the reboot — so the UI can show progress, then leave the actual swap to
// start.sh (which backs up the current install and can roll back).
//
//   - srcDir is the plugin source (resources commit their *_templ.go and db/queries,
//     so codegen is skipped and no templ/sqlc tooling is needed on-device).
//   - coreStageDir is the staged core payload (GetPendingUpdatePath(CorePkg)); its
//     core/ + sdk/ are the NEW core the .so must be ABI-matched to.
//   - pinned is the TARGET core version's dependency lock (from plugindeps.Fetch with
//     the staged core version) so the rebuilt .so resolves the same module
//     versions+hashes as the staged core and the cloud-built store plugins beside it.
//     Nil = build unpinned (lock unavailable/empty).
//
// The staged package is the current install tree (resources, assets/dist, plugin.json
// — all ABI-independent and unchanged) with plugin.so replaced by the freshly built
// one; start.sh's apply_pkg swaps the whole install dir for it.
func StageLocalPluginRebuild(srcDir, coreStageDir string, pinned []LockedGoModule) error {
	info, err := sdkutils.GetPluginInfoFromPath(srcDir)
	if err != nil {
		return err
	}

	// Only plugins that are actually installed can be staged for replacement (we
	// reuse their current install tree for the ABI-independent resources).
	currentInstall := GetInstallPath(info.Package)
	if !sdkutils.FsExists(currentInstall) {
		return fmt.Errorf("plugin %s is not installed", info.Package)
	}

	workdir := filepath.Join(sdkutils.PathTmpDir, "stage-rebuild", info.Package)
	defer os.RemoveAll(workdir)

	// Pin dependencies on a DISPOSABLE copy of the plugin source, never on srcDir
	// itself. PatchPluginDeps rewrites go.mod/go.sum in place (see build-plugins.go's
	// BuildPlugin, which does the same for the live core), and here the pin target is
	// the STAGED (not-yet-applied) core's lock — a DIFFERENT version than the one the
	// machine is still actually running. If this update is later cancelled or fails
	// before reboot, srcDir must be left exactly as it was: still buildable against
	// the current live core, not silently re-pinned to a core that never shipped.
	pinnedSrcDir := filepath.Join(workdir, "src")
	if err := sdkutils.FsCopyDir(srcDir, pinnedSrcDir, nil); err != nil {
		return fmt.Errorf("copy plugin source for pinned rebuild: %w", err)
	}
	if err := PatchPluginDeps(pinnedSrcDir, pinned); err != nil {
		return fmt.Errorf("pin plugin deps to staged core lock: %w", err)
	}

	// Compile plugin.so against the STAGED core (opts.AppDir), not the live one, from
	// the pinned copy so the .so resolves the same module versions+hashes as the
	// staged core and the cloud-built store plugins beside it.
	buildWorkdir := filepath.Join(workdir, "build")
	if err := BuildPluginSo(pinnedSrcDir, buildWorkdir, BuildOpts{
		SkipTemplates: true,
		SkipQueries:   true,
		AppDir:        coreStageDir,
	}); err != nil {
		return fmt.Errorf("build plugin.so against staged core: %w", err)
	}

	builtSo := filepath.Join(pinnedSrcDir, "plugin.so")
	if !sdkutils.FsExists(builtSo) {
		return fmt.Errorf("build produced no plugin.so")
	}

	// Assemble the staged install tree: current install + freshly built plugin.so.
	staged := GetPendingUpdatePath(info.Package)
	if err := os.RemoveAll(staged); err != nil {
		return err
	}
	if err := sdkutils.FsCopyDir(currentInstall, staged, nil); err != nil {
		return fmt.Errorf("assemble staged package: %w", err)
	}
	if err := sdkutils.FsCopyFile(builtSo, filepath.Join(staged, "plugin.so")); err != nil {
		return fmt.Errorf("stage plugin.so: %w", err)
	}

	fmt.Printf("StageLocalPluginRebuild: staged %s rebuilt against new core\n", info.Package)
	return nil
}
