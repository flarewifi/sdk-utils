package updates

import (
	"core/internal/rpc"
	"core/internal/rpc_flarewifi_v2"
	"core/utils/config"
	"core/utils/tags"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/Masterminds/semver/v3"
	sdkutils "github.com/flarehotspot/sdk-utils"
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
)

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

	srv, ctx := rpc.GetTwirpServiceAndCtx()
	params := rpc_flarewifi_v2.FetchLatestSoftwareReleaseRequest{
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

	log.Println("\nChecking software version:")
	sdkutils.PrettyPrint(&params)

	result, err := srv.FetchLatestSoftwareRelease(ctx, &params)
	if err != nil {
		return nil, ErrCheckUpdate
	}

	fmt.Printf("Software update check result: %+v\n", result)

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
				log.Println("Error downloading update file:", result.Error)
				downloadError.Store(&result.Error)
				return
			}

			downloadPercent.Store(int32(result.Percent))
			downloadedBytes.Store(result.Downloaded)
			totalSizeBytes.Store(result.TotalSize)
			prev := prevPercent.Load()

			if prev < int32(result.Percent) {
				log.Println("Download percent:", result.Percent)
			}
			prevPercent.Store(int32(result.Percent))
		}
	}()
}

// downloadFile downloads a file from FileURL to OutputPath with checksum verification
func downloadFile(params DownloadParams) (resultCh chan DownloadResult) {
	resultCh = make(chan DownloadResult)

	go func() {
		defer close(resultCh)

		log.Println("Downloading update file from", params.FileURL, "to", params.OutputPath)

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
				log.Printf("Checksum mismatch: expected %s, got %s", params.Checksum, actualChecksum)
				os.Remove(params.OutputPath)
				resultCh <- DownloadResult{Error: ErrChecksumMismatch}
				return
			}
			log.Println("Checksum verified successfully")
		}

		log.Println("Update file downloaded to", params.OutputPath)

		// Ensure the marker directory exists
		if err := sdkutils.FsEnsureDir(sdkutils.PathSystemUpdateDir); err != nil {
			log.Println("Warning: failed to create marker directory:", err)
		}

		// Create completion marker file
		markerPath := filepath.Join(sdkutils.PathSystemUpdateDir, downloadCompleteFile)
		markerContent := "complete"
		if params.IsSysupgrade {
			markerContent = "sysupgrade"
		}
		if err := os.WriteFile(markerPath, []byte(markerContent), 0644); err != nil {
			log.Println("Warning: failed to create download completion marker:", err)
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

func IsDownloaded() bool {
	markerPath := filepath.Join(sdkutils.PathSystemUpdateDir, downloadCompleteFile)
	return sdkutils.FsExists(markerPath)
}

func DownloadedBytes() int64 {
	return downloadedBytes.Load()
}

func TotalSizeBytes() int64 {
	return totalSizeBytes.Load()
}
