package updates

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v2"
	"core/utils/config"
	"core/utils/tags"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/Masterminds/semver/v3"
	sdkutils "github.com/flarewifi/sdk-utils"
)

var (
	// Errors
	ErrCheckUpdate      = errors.New("System update error: check update failed")
	ErrDownload         = errors.New("System update error: download failed")
	ErrExtract          = errors.New("System update error: extract failed")
	ErrChecksumMismatch = errors.New("System update error: checksum verification failed")

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

func CheckSoftwareReleaseUpdate(currentVersion *semver.Version) (*SoftwareReleaseUpdate, error) {
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

	srv, ctx := rpc.GetTwirpServiceAndCtx()
	params := rpc_flarewifi_v2.FetchLatestSoftwareReleaseRequest{
		MachineId:      machineID,
		DeviceModel:    release.DeviceModel,
		DeviceConfig:   release.DeviceConfig,
		CurrentVersion: currentVersion.String(),
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
				downloadError.Store(&result.Error)
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
