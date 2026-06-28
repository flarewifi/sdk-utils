//go:build !mono

package api

import (
	"context"
	machineuid "core/internal/modules/machine-uid"
	corerpc "core/internal/rpc"
	v3 "core/internal/rpc/rpc_flarewifi_v3"
	"core/utils/plugins"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sdkplugin "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
)

const (
	pluginBuildStatusOK         = "ok"
	pluginBuildStatusFailed     = "failed"
	pluginBuildStatusProcessing = "processing"

	pluginBuildPollInterval = 3 * time.Second
	pluginBuildPollTimeout  = 10 * time.Minute
)

// PluginBuildError carries a clean, user-facing reason for a failed server-side
// plugin build — e.g. a plugin the developer disabled, or a compiler message from
// the cloud — as opposed to an internal/transport error. The software-update UI
// shows Reason verbatim, so it must read as an explanation. Build/staging code
// returns this (wrapped with %w) whenever the cloud reported a specific reason via
// the build response's error_message; callers extract it with errors.As.
type PluginBuildError struct{ Reason string }

func (e *PluginBuildError) Error() string { return e.Reason }

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

	downloadURL, err := self.fetchPrebuiltPluginURL(pkg, version, coreVersion, nil)
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
func (self *PluginsMgr) fetchPrebuiltPluginURL(pkg, version, coreVersion string, emit progressEmitter, extraMetas ...*v3.InstalledMeta) (string, error) {
	// The prebuild RPCs live on the core-owned FlarehotspotService (flarehotspot.v2),
	// NOT the store plugin's store.v1 — core and plugin must keep separate proto
	// files so their descriptors don't collide in the same process.
	srv, ctx := corerpc.GetTwirpServiceAndCtx()
	_, machineID := machineuid.GetMachineUID()

	emit.call(sdkplugin.PluginInstallStageQueued, 15, "")
	resp, err := srv.RequestPluginBuild(ctx, &v3.RequestPluginBuildRequest{
		MachineId:   machineID,
		Package:     pkg,
		Version:     version,
		CoreVersion: coreVersion,
		// Declare the Go version of this running core. A plugin .so is ABI-locked to
		// the exact Go version of the core it loads into, so the cloud must build/serve
		// the cache entry for this go version. sdkutils.GO_VERSION is runtime.Version()
		// of this binary — i.e. the toolchain it was compiled with — the authoritative
		// ABI value, independent of when the machine last registered.
		GoVersion: sdkutils.GO_VERSION,
		// Report installed bundles so the build backstop resolves meta-member coverage
		// against the INSTALLED meta version, matching FetchLatestPluginReleaseByPackage.
		InstalledMetas: self.installedMetasReport(extraMetas...),
	})
	if err != nil {
		return "", fmt.Errorf("request build for %s: %w", pkg, err)
	}

	// Synchronous refusal: the cloud rejected the build outright (e.g. the plugin was
	// disabled by its developer) and returned a failed status with a specific reason
	// rather than enqueuing a bot build. Surface that reason — there is no build to
	// poll. Wrapped as PluginBuildError so the software-update UI can show WHY.
	if resp.GetBuildStatus() == pluginBuildStatusFailed {
		return "", &PluginBuildError{Reason: resp.GetErrorMessage()}
	}

	// Fast path: the cloud already had this (plugin, core version, platform) built
	// and returned the download URL immediately. Otherwise poll until ready.
	downloadURL := resp.GetDownloadUrl()
	if resp.GetBuildStatus() != pluginBuildStatusOK {
		downloadURL, err = self.awaitPluginBuild(srv, ctx, machineID, resp.GetBuildId(), emit)
		if err != nil {
			return "", err
		}
	}
	if downloadURL == "" {
		return "", fmt.Errorf("%s build completed without a download url", pkg)
	}
	emit.call(sdkplugin.PluginInstallStageDownloading, 75, "")
	return downloadURL, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// awaitPluginBuild polls the cloud build until it is ready, failed, or times out,
// returning the install-ready tarball URL on success.
func (self *PluginsMgr) awaitPluginBuild(srv v3.FlarehotspotService, ctx context.Context, machineID string, buildID int64, emit progressEmitter) (string, error) {
	deadline := time.Now().Add(pluginBuildPollTimeout)
	// The cloud reports only coarse states, so ramp a synthetic percent within the
	// build phase (30→70) so a long compile still shows forward motion.
	buildPercent := 30
	for {
		st, err := srv.GetPluginBuildStatus(ctx, &v3.GetPluginBuildStatusRequest{
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
			// Carry the cloud's reason verbatim for the UI; fall back to the build id
			// when the server gave no message (so the failure is still traceable).
			reason := st.GetErrorMessage()
			if reason == "" {
				reason = fmt.Sprintf("server build %d failed", buildID)
			}
			return "", &PluginBuildError{Reason: reason}
		case pluginBuildStatusProcessing:
			emit.call(sdkplugin.PluginInstallStageBuilding, buildPercent, "")
			if buildPercent < 70 {
				buildPercent += 5
			}
		default: // pending: enqueued, waiting for a binary bot.
			emit.call(sdkplugin.PluginInstallStageQueued, 20, "")
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
