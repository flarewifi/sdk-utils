//go:build !mono

package api

import (
	"context"
	machineuid "core/internal/modules/machine-uid"
	corerpc "core/internal/rpc"
	v2 "core/internal/rpc/rpc_flarewifi_v2"
	"core/utils/plugins"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	pluginBuildStatusOK     = "ok"
	pluginBuildStatusFailed = "failed"

	pluginBuildPollInterval = 3 * time.Second
	pluginBuildPollTimeout  = 10 * time.Minute
)

// StagePluginUpdate requests a server-side prebuild of a store plugin, downloads
// the install-ready tarball, and extracts it into the unified staging directory
// (data/storage/system/updates/{pkg}) WITHOUT touching the live install. The
// non-mono boot script (start.sh) overlays it onto plugins/installed/{pkg}
// on the next restart.
//
// A Go plugin .so is ABI-locked to the core build it loads into, so the plugin is
// compiled in the cloud against coreVersion — the core version it will be applied
// WITH. Pass coreVersion == "" for a plugin-only update (the cloud builds against
// the machine's currently-registered core version); pass the TARGET core version
// when staging plugins alongside a core update (model A: core + plugins applied
// together in one boot).
//
// version may be empty to stage the latest release; it is resolved to a concrete
// semver before requesting the build. Meta bundles are not supported (no zip).
func (self *PluginsMgr) StagePluginUpdate(pkg, version, coreVersion string) error {
	// RequestPluginBuild needs a concrete semver, so resolve "latest" first.
	if version == "" {
		rel, err := self.fetchStoreRelease(pkg, "")
		if err != nil {
			return fmt.Errorf("StagePluginUpdate: resolve latest %s: %w", pkg, err)
		}
		if rel.IsMeta {
			return fmt.Errorf("StagePluginUpdate: %q is a meta bundle and cannot be updated this way", pkg)
		}
		version = rel.Version
	}

	downloadURL, err := self.fetchPrebuiltPluginURL(pkg, version, coreVersion)
	if err != nil {
		return fmt.Errorf("StagePluginUpdate: %w", err)
	}

	if err := stagePackageFromURL(pkg, downloadURL); err != nil {
		return fmt.Errorf("StagePluginUpdate: stage %s: %w", pkg, err)
	}
	return nil
}

// fetchPrebuiltPluginURL requests a server-side build of a store plugin and
// returns the install-ready tarball URL, polling until the build is done. A Go
// plugin .so is ABI-locked to the exact core build it loads into, so the cloud
// compiles against coreVersion ("" = the machine's currently-registered core
// version) for this machine's platform. version must be a concrete semver.
//
// The mono twin of this method always errors: mono machines statically link
// plugins at core-build time, so there is nothing a prebuilt .so could load into.
func (self *PluginsMgr) fetchPrebuiltPluginURL(pkg, version, coreVersion string) (string, error) {
	// The prebuild RPCs live on the core-owned FlarehotspotService (flarehotspot.v2),
	// NOT the store plugin's store.v1 — core and plugin must keep separate proto
	// files so their descriptors don't collide in the same process.
	srv, ctx := corerpc.GetTwirpServiceAndCtx()
	_, machineID := machineuid.GetMachineUID()

	resp, err := srv.RequestPluginBuild(ctx, &v2.RequestPluginBuildRequest{
		MachineId:   machineID,
		Package:     pkg,
		Version:     version,
		CoreVersion: coreVersion,
	})
	if err != nil {
		return "", fmt.Errorf("request build for %s: %w", pkg, err)
	}

	// Fast path: the cloud already had this (plugin, core version, platform) built
	// and returned the download URL immediately. Otherwise poll until ready.
	downloadURL := resp.GetDownloadUrl()
	if resp.GetBuildStatus() != pluginBuildStatusOK {
		downloadURL, err = self.awaitPluginBuild(srv, ctx, machineID, resp.GetBuildId())
		if err != nil {
			return "", err
		}
	}
	if downloadURL == "" {
		return "", fmt.Errorf("%s build completed without a download url", pkg)
	}
	return downloadURL, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// awaitPluginBuild polls the cloud build until it is ready, failed, or times out,
// returning the install-ready tarball URL on success.
func (self *PluginsMgr) awaitPluginBuild(srv v2.FlarehotspotService, ctx context.Context, machineID string, buildID int64) (string, error) {
	deadline := time.Now().Add(pluginBuildPollTimeout)
	for {
		st, err := srv.GetPluginBuildStatus(ctx, &v2.GetPluginBuildStatusRequest{
			MachineId: machineID,
			BuildId:   buildID,
		})
		if err != nil {
			return "", fmt.Errorf("poll build %d: %w", buildID, err)
		}

		switch st.GetBuildStatus() {
		case pluginBuildStatusOK:
			return st.GetDownloadUrl(), nil
		case pluginBuildStatusFailed:
			return "", fmt.Errorf("server build %d failed: %s", buildID, st.GetErrorMessage())
		}

		if time.Now().After(deadline) {
			return "", fmt.Errorf("server build %d timed out after %s", buildID, pluginBuildPollTimeout)
		}
		time.Sleep(pluginBuildPollInterval)
	}
}

// stagePackageFromURL downloads an install-ready package tarball and extracts it
// into the package's unified staging dir, replacing any previous staging. Validates
// the extracted tree carries a plugin.json so a corrupt download is never staged.
func stagePackageFromURL(pkg, url string) error {
	dest := plugins.GetPendingUpdatePath(pkg)
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := sdkutils.FsEnsureDir(dest); err != nil {
		return err
	}

	tmpFile := filepath.Join(sdkutils.PathTmpDir, "plugin-update-"+pkg+"-"+sdkutils.RandomStr(8)+".tar.gz")
	if err := sdkutils.FsEnsureDir(filepath.Dir(tmpFile)); err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	if err := sdkutils.Download(url, tmpFile); err != nil {
		os.RemoveAll(dest)
		return fmt.Errorf("download: %w", err)
	}
	if err := sdkutils.Untar(tmpFile, dest); err != nil {
		os.RemoveAll(dest)
		return fmt.Errorf("extract: %w", err)
	}
	if err := plugins.ValidateInstallPath(dest); err != nil {
		os.RemoveAll(dest)
		return fmt.Errorf("staged package is missing plugin.json: %w", err)
	}
	return nil
}
