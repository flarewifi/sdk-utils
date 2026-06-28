//go:build !mono

// Non-mono local software-release apply.
//
// A manually uploaded software-release tarball is a COMPLETE, ABI-matched build: it
// bundles bin/flare, the rebuilt core, plugins/installed (every plugin already
// compiled against this exact core), and the local plugin sources. So unlike the
// online StageSystemUpdate — which downloads only the shared core and rebuilds every
// plugin (store plugins in the cloud, local plugins on-device) — the local apply is
// fully OFFLINE: stage the tarball as the core package and let start.sh's core
// overlay (which includes plugins/installed) carry the whole set onto the app on the
// next reboot. No cloud round-trip, no recompile, no product-version stamping (the
// tarball already carries its own core/product.json).
package updates

import (
	"core/internal/api"
	"core/utils/plugins"
	"errors"
	"fmt"
	"os"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// StageLocalSoftwareRelease stages an already-saved software-release tarball
// (srcPath) for apply-on-reboot on a non-mono machine. On success IsDownloaded()
// becomes true (the .staged_complete marker is written) and the caller routes into
// the existing download-done → reboot flow.
func StageLocalSoftwareRelease(g *api.CoreGlobals, srcPath string) error {
	// Start from a clean staging root so a previous partial/aborted attempt can't
	// leak into this one.
	if err := sdkutils.FsEmptyDir(sdkutils.PathSystemUpdateDir); err != nil {
		return err
	}

	// Extract the release into the staged core package dir (updates/com.flarego.core).
	coreDest := plugins.GetPendingUpdatePath(plugins.CorePkg)
	if err := os.RemoveAll(coreDest); err != nil {
		return err
	}
	if err := sdkutils.FsEnsureDir(coreDest); err != nil {
		return err
	}
	if err := sdkutils.Untar(srcPath, coreDest); err != nil {
		return ErrExtract
	}
	if !plugins.HasPendingCoreUpdate() {
		return errors.New("uploaded release is missing bin/flare (corrupt archive)")
	}

	// Drop bundled sources for plugins NOT registered local on this device, then
	// normalize devel source paths — the same routing the online flow applies so a
	// store plugin's bundled source isn't mistaken for a local one to recompile.
	if err := pruneNonLocalPluginSources(coreDest); err != nil {
		return fmt.Errorf("prune non-local plugin sources: %w", err)
	}
	if err := convertDevelPluginsToLocal(); err != nil {
		return fmt.Errorf("convert devel plugins to local: %w", err)
	}

	// Commit last: start.sh applies the staged set atomically the moment this marker
	// exists, so a crash mid-staging leaves an incomplete set that start.sh discards.
	if err := plugins.MarkStagedComplete(); err != nil {
		return fmt.Errorf("write staged-complete marker: %w", err)
	}

	return nil
}
