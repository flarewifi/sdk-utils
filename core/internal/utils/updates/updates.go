package updates

import (
	rpc "core/internal/rpc"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"tools/config"
	"tools/tags"

	"github.com/Masterminds/semver/v3"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

var (
	// Errors
	ErrCheckUpdate      = errors.New("System update error: check update failed")
	ErrDownload         = errors.New("System update error: download failed")
	ErrExtract          = errors.New("System update error: extract failed")
	ErrChecksumMismatch = errors.New("System update error: checksum verification failed")

	osReleaseFile   = "/etc/os_release.json"
	downloading     atomic.Bool
	downloadPercent atomic.Int32
	prevPercent     atomic.Int32
	downloadError   atomic.Pointer[error]
)

type SoftwareReleaseUpdate struct {
	Version             *semver.Version
	ReleseFileURL       string
	ReleaseFileChecksum string
	ReleaseNotes        string
	HasUpdate           bool
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
	params := rpc.FetchLatestSoftwareReleaseRequest{
		CurrentVersion: currentVersion.String(),
		BrandId:        release.BrandId,
		Os:             strings.ToLower(release.Os),
		OsVersion:      release.OsVersion,
		OsTarget:       release.OsTarget,
		OsArch:         release.OsArch,
		OsProfile:      release.OsProfile,
		OsConfig:       release.OsConfig,
		GoVersion:      sdkutils.GO_VERSION,
		GoArch:         sdkutils.GOARCH,
		IsMono:         tags.HasGoTag("mono"),
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
	}

	return update, nil
}

type DownloadResult struct {
	Percent int
	Error   error
}

func DownloadSoftwareRelease(releaseFileUrl string, md5sum string) {
	isDownloading := downloading.Load()

	if isDownloading {
		return
	}

	downloading.Store(true)
	downloadPercent.Store(0)
	prevPercent.Store(0)

	go func() {
		defer downloading.Store(false)
		defer downloadPercent.Store(0)
		defer prevPercent.Store(0)

		if err := sdkutils.FsEmptyDir(sdkutils.PathPluginUpdatesDir); err != nil {
			downloadError.Store(&err)
			return
		}

		ch := downloadSystemFile(releaseFileUrl, md5sum)

		for result := range ch {
			if result.Error != nil {
				log.Println("Error downloading system files:", result.Error)
				downloadError.Store(&result.Error)
				return
			}

			downloadPercent.Store(int32(result.Percent))
			prev := prevPercent.Load()

			if prev < int32(result.Percent) {
				log.Println("Download percent:", result.Percent)
			}
			prevPercent.Store(int32(result.Percent))
		}
	}()
}

func downloadSystemFile(fileURL string, expectedChecksum string) (resultCh chan DownloadResult) {
	resultCh = make(chan DownloadResult)
	downloadFilePath := filepath.Join(sdkutils.PathTmpDir, "system", "update", filepath.Base(fileURL))

	// Create download options with checksum verification
	opts := &sdkutils.DownloadChOpts{
		Md5Checksum: expectedChecksum,
	}
	percentCh, errCh := sdkutils.DownloadCh(fileURL, downloadFilePath, opts)

	go func() {
		defer close(resultCh)
		defer os.RemoveAll(downloadFilePath)

		for {
			result := DownloadResult{}
			select {
			case percent := <-percentCh:
				// do something with the percentage
				result.Percent = percent
				resultCh <- result

			case err, ok := <-errCh:
				if !ok {
					return
				}

				if err != nil {
					fmt.Println("Error downloading software update", err)
					// Check if it's a checksum error
					if errors.Is(err, sdkutils.ErrChecksumVerificationFailed) {
						result.Error = ErrChecksumMismatch
					} else {
						result.Error = ErrDownload
					}
					resultCh <- result
					return
				}

				log.Println("Extracting updates files to", sdkutils.PathSystemUpdateDir)
				err = sdkutils.FsExtract(downloadFilePath, sdkutils.PathSystemUpdateDir)
				if err != nil {
					fmt.Println("Error extracting update files", err)
					result.Error = ErrExtract
					resultCh <- result
					return
				}

				result.Percent = 100
				resultCh <- result
				return
			}
		}
	}()

	return resultCh
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
	return sdkutils.FsExists(sdkutils.PathSystemUpdateDir)
}
