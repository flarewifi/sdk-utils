package updates

import (
	rpc "core/internal/rpc"
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
	downloadFilePath := filepath.Join(sdkutils.PathSystemUpdateDir, filepath.Base(fileURL))

	go func() {
		defer close(resultCh)

		log.Println("Downloading compressed update file from", fileURL)

		// Ensure the destination directory exists
		if err := sdkutils.FsEnsureDir(sdkutils.PathSystemUpdateDir); err != nil {
			result := DownloadResult{Error: ErrDownload}
			resultCh <- result
			return
		}

		// Make HTTP request
		resp, err := http.Get(fileURL)
		if err != nil {
			result := DownloadResult{Error: ErrDownload}
			resultCh <- result
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			result := DownloadResult{Error: ErrDownload}
			resultCh <- result
			return
		}

		totalSize := resp.ContentLength
		if totalSize <= 0 {
			result := DownloadResult{Error: ErrDownload}
			resultCh <- result
			return
		}

		// Create the output file
		file, err := os.Create(downloadFilePath)
		if err != nil {
			result := DownloadResult{Error: ErrDownload}
			resultCh <- result
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
					result := DownloadResult{Error: ErrDownload}
					resultCh <- result
					return
				}
				downloaded += int64(n)
				currentPercent := int((downloaded * 100) / totalSize)
				if currentPercent != lastPercent {
					resultCh <- DownloadResult{Percent: currentPercent}
					lastPercent = currentPercent
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				result := DownloadResult{Error: ErrDownload}
				resultCh <- result
				return
			}
		}

		// Verify checksum after download completes
		if expectedChecksum != "" {
			actualChecksum := base64.StdEncoding.EncodeToString(hasher.Sum(nil))
			if actualChecksum != expectedChecksum {
				log.Printf("Checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
				os.Remove(downloadFilePath)
				result := DownloadResult{Error: ErrChecksumMismatch}
				resultCh <- result
				return
			}
			log.Println("Checksum verified successfully")
		}

		log.Println("Compressed update file downloaded to", downloadFilePath)
		result := DownloadResult{Percent: 100}
		resultCh <- result
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
