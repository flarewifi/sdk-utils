package updates

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v3"
	"core/utils/config"
	"core/utils/product"
	"core/utils/tags"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Masterminds/semver/v3"
	sdkutils "github.com/flarewifi/sdk-utils"
)

var (
	// Errors
	ErrCheckUpdate      = errors.New("System update error: check update failed")
	ErrDownload         = errors.New("System update error: download failed")
	ErrExtract          = errors.New("System update error: extract failed")
	ErrChecksumMismatch = errors.New("System update error: checksum verification failed")
	// ErrUpdateCancelled is returned by the staging flow when the admin chose to
	// skip the whole update at the plugin-build-failure confirmation gate. It is a
	// clean, user-initiated stop — NOT surfaced as a download error — so the staging
	// goroutine treats it as a quiet exit (see StageSystemUpdate/StagePluginsUpdate).
	ErrUpdateCancelled = errors.New("System update cancelled by admin")

	osReleaseFile        = "/etc/os_release.json"
	downloadCompleteFile = ".dl_software_update_complete"
	downloading          atomic.Bool
	downloadPercent      atomic.Int32
	prevPercent          atomic.Int32
	downloadError        atomic.Pointer[error]
	downloadedBytes      atomic.Int64
	totalSizeBytes       atomic.Int64
	// updatePhase identifies which stage of an in-flight upgrade is running so the
	// download page can label it correctly. A non-mono upgrade first downloads the
	// core tarball (PhaseDownloading) and then stages/recompiles plugins against it
	// (PhaseCompiling); the byte counters above only advance during the download.
	// Reset to PhaseDownloading at the start of every download/stage.
	updatePhase atomic.Int32
	// pluginUpdateApplied marks a plugin-only upgrade that completed WITHOUT staging
	// anything for reboot (its only change — a meta-bundle version bump — was applied
	// live). It lets the download flow finish without prompting for a reboot. Reset
	// at the start of every stage/download.
	pluginUpdateApplied atomic.Bool

	// Plugin-build-failure confirmation gate. When one or more plugins fail to build
	// during staging, the staging goroutine PAUSES here and waits for the admin to
	// choose continue (skip the failed plugins, apply the rest) or cancel (skip the
	// whole update). awaitingConfirm gates the download-status page into rendering the
	// dialog; confirmCh delivers the admin's choice back to the paused goroutine.
	awaitingConfirm atomic.Bool
	confirmMu       sync.Mutex
	confirmFailed   []PluginBuildFailure // plugins that failed to build (shown in the dialog)
	confirmCh       chan bool            // true = continue, false = cancel; buffered(1)

	// cancelRequested signals an admin-initiated cancel of an in-flight
	// download/stage. It is checked at safe checkpoints only — between downloaded
	// chunks and between individual plugin builds — never mid-flash, so it can only
	// ever abort work that is still fully reversible (see ErrUpdateCancelled).
	// Reset at the start of every new download/stage.
	cancelRequested atomic.Bool

	// lastStagedPlugins and lastSkippedPlugins record what the most recently
	// completed stage actually did, for the download-done page's "what changed"
	// summary. Both hold a []T behind atomic.Value (never mutated in place — always
	// swapped for a new slice) since staging is strictly single-goroutine (the
	// downloading flag rules out a concurrent stage), so no locking is needed.
	// Reset at the start of every new download/stage; mono never populates either
	// (it has no per-plugin staging step — system-update.go is !mono only).
	lastStagedPlugins  atomic.Value
	lastSkippedPlugins atomic.Value
)

// UpdatePhase identifies which stage of an in-flight upgrade is running, so the
// download page can show "Downloading updates" versus "Compiling plugins".
type UpdatePhase int32

const (
	// PhaseDownloading: fetching the core tarball / sysupgrade image from the cloud.
	// The byte counters are meaningful in this phase. Default for mono (whose whole
	// flow is a single tarball download) and for the start of every non-mono stage.
	PhaseDownloading UpdatePhase = iota
	// PhaseCompiling: staging plugins — store plugins built in the cloud and on-device
	// recompiles against the staged core. No byte progress; only the percent advances.
	PhaseCompiling
)

// setPhase records the current upgrade stage. Called by the non-mono staging flow;
// mono never leaves PhaseDownloading.
func setPhase(p UpdatePhase) {
	updatePhase.Store(int32(p))
}

// CurrentPhase reports which stage of the in-flight upgrade is running. The download
// page uses it to label the progress UI. See UpdatePhase.
func CurrentPhase() UpdatePhase {
	return UpdatePhase(updatePhase.Load())
}

type SoftwareReleaseUpdate struct {
	Version             *semver.Version
	ReleseFileURL       string
	ReleaseFileChecksum string
	ReleaseNotes        string
	HasUpdate           bool
	IsSysupgrade        bool
	ForceUpdate         bool
}

func CheckSoftwareReleaseUpdate() (*SoftwareReleaseUpdate, error) {
	release, err := sdkutils.ReadOsRelease(osReleaseFile)
	if err != nil {
		return nil, err
	}

	cfg, err := config.ReadApplicationConfig()
	if err != nil {
		return nil, err
	}

	// The server gates beta/development releases to tester-owned machines, looking up
	// the owner by machine_id from the request body. Omitting it makes the server treat
	// the machine as a non-tester and silently downgrade the requested channel to
	// "stable", hiding every beta/development release. Always send it.
	_, machineID := machineuid.GetMachineUID()

	// The machine's ABI core version (core/plugin.json). Best-effort: the server uses
	// ProductVersion for update-eligibility, so an unreadable core version doesn't block
	// the check — it only omits the informational ABI value.
	coreVersion := ""
	if coreInfo, cerr := sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir); cerr == nil {
		coreVersion = coreInfo.Version
	}

	srv, ctx := rpc.GetTwirpServiceAndCtx()
	params := rpc_flarewifi_v3.FetchLatestSoftwareReleaseRequest{
		MachineId:    machineID,
		DeviceModel:  release.DeviceModel,
		DeviceConfig: release.DeviceConfig,
		// CoreVersion is the ABI identity (core/plugin.json). ProductVersion is the
		// per-partner release lineage the server compares for update-eligibility;
		// product.Version falls back to the core version on builds that were never
		// stamped (older images / dev).
		CoreVersion:    coreVersion,
		ProductVersion: product.Version(),
		BrandId:        release.BrandId,
		Os:             strings.ToLower(release.Os),
		OsVersion:      release.OsVersion,
		OsTarget:       release.OsTarget,
		OsArch:         release.OsArch,
		OsProfile:      release.OsProfile,
		GoVersion:      sdkutils.GO_VERSION,
		GoArch:         sdkutils.GOARCH,
		Monolythic:     tags.HasGoTag("mono"),
		Channel:        strings.ToLower(cfg.Channel),
	}

	sdkutils.PrettyPrint(&params)

	result, err := srv.FetchLatestSoftwareRelease(ctx, &params)
	if err != nil {
		return nil, ErrCheckUpdate
	}

	if !result.HasUpdate {
		return &SoftwareReleaseUpdate{HasUpdate: false}, nil
	}

	latestVersion, err := semver.NewVersion(result.Version)
	if err != nil {
		return nil, err
	}

	update := &SoftwareReleaseUpdate{
		HasUpdate:           result.HasUpdate,
		Version:             latestVersion,
		ReleseFileURL:       result.FileUrl,
		ReleaseFileChecksum: result.FileChecksum,
		ReleaseNotes:        result.ReleaseNotes,
		IsSysupgrade:        result.IsSysupgrade,
		ForceUpdate:         result.ForceUpdate,
	}

	return update, nil
}

type DownloadResult struct {
	Percent    int
	Downloaded int64 // bytes downloaded so far
	TotalSize  int64 // total file size in bytes
	Error      error
}

// DownloadParams contains parameters for downloading a software update
type DownloadParams struct {
	FileURL      string // URL to download from
	IsSysupgrade bool   // Whether this is a firmware sysupgrade
	Checksum     string // MD5 checksum for verification (base64 encoded)
	OutputPath   string // Destination file path
}

// DownloadSoftwareUpdate downloads a software update file to the specified output path.
func DownloadSoftwareUpdate(params DownloadParams) {
	if downloading.Load() {
		return
	}

	downloading.Store(true)
	downloadPercent.Store(0)
	prevPercent.Store(0)
	downloadedBytes.Store(0)
	totalSizeBytes.Store(0)
	downloadError.Store(nil) // Clear any previous download error
	cancelRequested.Store(false)
	lastStagedPlugins.Store([]StagedComponent{})
	lastSkippedPlugins.Store([]PluginBuildFailure{})
	setPhase(PhaseDownloading)

	go func() {
		defer downloading.Store(false)
		defer downloadPercent.Store(0)
		defer prevPercent.Store(0)
		defer downloadedBytes.Store(0)
		defer totalSizeBytes.Store(0)

		// Clean up before starting new download to free up space
		// Always remove sysupgrade file and clean all update directories
		RemoveSysupgradeFile()

		// Clean up system updates directory
		if err := sdkutils.FsEmptyDir(sdkutils.PathSystemUpdateDir); err != nil {
			downloadError.Store(&err)
			return
		}

		// Clean up plugin updates directory
		if err := sdkutils.FsEmptyDir(sdkutils.PathPluginUpdatesDir); err != nil {
			downloadError.Store(&err)
			return
		}

		ch := downloadFile(params)

		for result := range ch {
			if result.Error != nil {
				// A clean, admin-initiated cancel is not an error — quiet exit so the
				// download page redirects to the updates index (the cancel endpoint
				// does the redirect) instead of showing a scary download error. Mirrors
				// the plugin-build-failure gate's ErrUpdateCancelled handling.
				if !errors.Is(result.Error, ErrUpdateCancelled) {
					downloadError.Store(&result.Error)
				}
				return
			}

			downloadPercent.Store(int32(result.Percent))
			downloadedBytes.Store(result.Downloaded)
			totalSizeBytes.Store(result.TotalSize)
			prevPercent.Store(int32(result.Percent))
		}
	}()
}

// downloadFile downloads a file from FileURL to OutputPath with checksum verification
func downloadFile(params DownloadParams) (resultCh chan DownloadResult) {
	resultCh = make(chan DownloadResult)

	go func() {
		defer close(resultCh)

		// Ensure the destination directory exists
		outputDir := filepath.Dir(params.OutputPath)
		if err := sdkutils.FsEnsureDir(outputDir); err != nil {
			resultCh <- DownloadResult{Error: ErrDownload}
			return
		}

		// Make HTTP request
		resp, err := http.Get(params.FileURL)
		if err != nil {
			resultCh <- DownloadResult{Error: ErrDownload}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			resultCh <- DownloadResult{Error: ErrDownload}
			return
		}

		totalSize := resp.ContentLength
		if totalSize <= 0 {
			resultCh <- DownloadResult{Error: ErrDownload}
			return
		}

		// Create the output file
		file, err := os.Create(params.OutputPath)
		if err != nil {
			resultCh <- DownloadResult{Error: ErrDownload}
			return
		}
		defer file.Close()

		// Create a hash writer for checksum verification
		hasher := md5.New()
		writer := io.MultiWriter(file, hasher)

		// Download with progress tracking
		downloaded := int64(0)
		lastPercent := -1
		buf := make([]byte, 32*1024)

		for {
			if cancelRequested.Load() {
				os.Remove(params.OutputPath)
				resultCh <- DownloadResult{Error: ErrUpdateCancelled}
				return
			}

			n, err := resp.Body.Read(buf)
			if n > 0 {
				_, writeErr := writer.Write(buf[:n])
				if writeErr != nil {
					os.Remove(params.OutputPath)
					resultCh <- DownloadResult{Error: ErrDownload}
					return
				}
				downloaded += int64(n)
				currentPercent := int((downloaded * 100) / totalSize)
				if currentPercent != lastPercent {
					resultCh <- DownloadResult{
						Percent:    currentPercent,
						Downloaded: downloaded,
						TotalSize:  totalSize,
					}
					lastPercent = currentPercent
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				os.Remove(params.OutputPath)
				resultCh <- DownloadResult{Error: ErrDownload}
				return
			}
		}

		// Verify checksum after download completes
		if params.Checksum != "" {
			actualChecksum := base64.StdEncoding.EncodeToString(hasher.Sum(nil))
			if actualChecksum != params.Checksum {
				os.Remove(params.OutputPath)
				resultCh <- DownloadResult{Error: ErrChecksumMismatch}
				return
			}
		}

		// For sysupgrade files, use the shared finalization path
		// This validates compatibility and creates the marker file
		if params.IsSysupgrade {
			if err := FinalizeSysupgrade(); err != nil {
				resultCh <- DownloadResult{Error: err}
				return
			}
		} else {
			// For regular updates, create the completion marker
			sdkutils.FsEnsureDir(sdkutils.PathSystemUpdateDir)
			markerPath := filepath.Join(sdkutils.PathSystemUpdateDir, downloadCompleteFile)
			os.WriteFile(markerPath, []byte("complete"), 0644)
		}

		resultCh <- DownloadResult{
			Percent:    100,
			Downloaded: totalSize,
			TotalSize:  totalSize,
		}
	}()

	return resultCh
}

// GetUpdateOutputPath returns the appropriate output path based on update type
func GetUpdateOutputPath(fileUrl string, isSysupgrade bool) string {
	if isSysupgrade {
		return GetSysupgradePath()
	}
	return filepath.Join(sdkutils.PathSystemUpdateDir, filepath.Base(fileUrl))
}

func IsDownloading() bool {
	return downloading.Load()
}

// RequestCancelDownload asks an in-flight download/stage to stop at its next safe
// checkpoint (see cancelRequested). A no-op if nothing is currently downloading —
// cancelRequested is cleared at the start of every new run regardless, so a stale
// request can never bleed into a later one. Callers should follow up with
// WaitForStagingToStop so the goroutine has actually unwound (flag cleared, partial
// state discarded) before redirecting.
func RequestCancelDownload() {
	if downloading.Load() {
		cancelRequested.Store(true)
	}
}

// PluginUpdateApplied reports whether the last plugin-only upgrade finished by
// applying its change live (a meta-bundle version bump) with nothing staged for
// reboot. The download flow uses this to show a success state instead of a reboot
// prompt. See stagePluginsUpdate.
func PluginUpdateApplied() bool {
	return pluginUpdateApplied.Load()
}

func DownloadPercent() int32 {
	return downloadPercent.Load()
}

func DownloadError() error {
	v := downloadError.Load()
	if v == nil {
		return nil
	}

	return *v
}

// IsDownloaded reports whether a system update has finished staging/downloading
// and is ready to apply. Its definition is build-tag specific: mono checks the
// single-tarball download marker (system-update_mono.go); non-mono checks the
// staged-overlay completion marker (system-update.go).

func DownloadedBytes() int64 {
	return downloadedBytes.Load()
}

func TotalSizeBytes() int64 {
	return totalSizeBytes.Load()
}

// AwaitingPluginConfirm reports whether staging is paused waiting for the admin to
// resolve the plugin-build-failure dialog (continue or cancel). The download-status
// page uses it to render the dialog instead of the progress bar.
func AwaitingPluginConfirm() bool {
	return awaitingConfirm.Load()
}

// PluginBuildFailure is one row of the build-failure confirmation dialog: the
// package that failed and a clean, human-readable reason (the cloud's message for a
// disabled plugin or a compile error, or the staging error for a local plugin).
type PluginBuildFailure struct {
	Package string
	Reason  string
}

// StagedComponent is one plugin (store or local) that the most recently completed
// stage actually rebuilt and staged for the next reboot. Used by the download-done
// page's "updated components" summary — see LastStagedPlugins.
type StagedComponent struct {
	Package string
	Name    string
}

// recordStagedPlugin appends a successfully staged plugin to the run's summary.
// Called only from the staging goroutine (system-update.go), which is always
// single-flight, so a plain load-copy-store is race-free without extra locking.
func recordStagedPlugin(c StagedComponent) {
	cur, _ := lastStagedPlugins.Load().([]StagedComponent)
	lastStagedPlugins.Store(append(append([]StagedComponent(nil), cur...), c))
}

// LastStagedPlugins returns the plugins the most recently completed stage actually
// rebuilt and staged for reboot (package + display name). Empty if none were staged,
// staging is still in progress, or on mono (no per-plugin staging step). Reset at
// the start of every new download/stage.
func LastStagedPlugins() []StagedComponent {
	v, _ := lastStagedPlugins.Load().([]StagedComponent)
	return v
}

// LastSkippedPlugins returns the plugins the most recently completed stage skipped
// because their build failed and the admin chose to continue past the confirmation
// gate (see confirmOrCancelOnFailures). Empty if none were skipped, or on mono.
// Reset at the start of every new download/stage.
func LastSkippedPlugins() []PluginBuildFailure {
	v, _ := lastSkippedPlugins.Load().([]PluginBuildFailure)
	return v
}

// FailedPluginBuilds returns the plugins that failed to build (package + reason),
// for display in the confirmation dialog. Returns a copy so callers can't mutate the
// gate's state.
func FailedPluginBuilds() []PluginBuildFailure {
	confirmMu.Lock()
	defer confirmMu.Unlock()
	return append([]PluginBuildFailure(nil), confirmFailed...)
}

// waitForPluginFailureDecision pauses the staging goroutine until the admin resolves
// the dialog, returning true to continue (skip the failed plugins, apply the rest)
// or false to cancel the whole update. It publishes the failed list and arms the
// decision channel before blocking, and clears the gate state on return. Called only
// from the staging goroutine, and only when failed is non-empty.
func waitForPluginFailureDecision(failed []PluginBuildFailure) bool {
	confirmMu.Lock()
	confirmFailed = append([]PluginBuildFailure(nil), failed...)
	confirmCh = make(chan bool, 1)
	ch := confirmCh
	confirmMu.Unlock()

	awaitingConfirm.Store(true)
	proceed := <-ch
	awaitingConfirm.Store(false)

	confirmMu.Lock()
	confirmFailed = nil
	confirmCh = nil
	confirmMu.Unlock()
	return proceed
}

// WaitForStagingToStop blocks until no staging/download is in flight (downloading
// flag cleared) or the timeout elapses. A caller that just cancelled uses it so the
// staging goroutine has a chance to finish its teardown — flag cleared, staged set
// discarded — BEFORE redirecting. The timeout is NOT a guarantee: cancellation is
// cooperative (checked only between plugin builds, see cancelRequested), so an
// on-device local-plugin recompile already in flight can legitimately outlast it.
// Callers MUST check IsDownloading() after this returns rather than assume success —
// redirecting to the updates index while it is still true bounces the admin straight
// back to the progress page (which, for a core update, would even auto-restart
// staging) — the "stuck in Compiling Plugins" symptom.
func WaitForStagingToStop(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for downloading.Load() {
		if !time.Now().Before(deadline) {
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
}

// ResolvePluginFailureDecision delivers the admin's choice to the paused staging
// goroutine: proceed=true continues (skip the failed plugins), proceed=false cancels
// the whole update. Safe to call when nothing is waiting (no-op) and safe against a
// double click — the channel is buffered(1) and the send is non-blocking, so a second
// resolve is dropped rather than blocking the request.
func ResolvePluginFailureDecision(proceed bool) {
	confirmMu.Lock()
	ch := confirmCh
	confirmMu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- proceed:
	default:
	}
}

