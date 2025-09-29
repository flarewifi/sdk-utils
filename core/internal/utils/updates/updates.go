package updates

import (
	rpc "core/internal/rpc"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/Masterminds/semver/v3"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

var (
	// Errors
	ErrCheckUpdate = errors.New("System update error: check update failed")
	ErrDownload    = errors.New("System update error: download failed")
	ErrExtract     = errors.New("System update error: extract failed")

	downloading     atomic.Bool
	downloadCh      atomic.Value
	downloadPercent atomic.Int32
	downloadError   atomic.Pointer[error]
)

type CoreReleaseUpdate struct {
	Version        *semver.Version
	CoreZipFileUrl string
	ArchBinFileUrl string
	HasUpdate      bool
}

func CheckCoreReleaseUpdate(currentVersion *semver.Version) (*CoreReleaseUpdate, error) {
	srv, ctx := rpc.GetCoreTwirpServiceAndCtx()

	result, err := srv.FetchLatestCoreRelease(ctx, &rpc.FetchLatestCoreReleaseRequest{
		CurrentCoreVersion: currentVersion.String(),
		GoVersion:          sdkutils.GO_VERSION,
		GoArch:             sdkutils.GOARCH,
	})
	if err != nil {
		return nil, ErrCheckUpdate
	}

	if !result.HasNewUpdate {
		return &CoreReleaseUpdate{HasUpdate: false}, nil
	}

	latestVersion, err := semver.NewVersion(fmt.Sprintf("%d.%d.%d", result.GetMajor(), result.GetMinor(), result.GetPatch()))
	if err != nil {
		return nil, err
	}

	update := &CoreReleaseUpdate{
		HasUpdate:      true,
		Version:        latestVersion,
		CoreZipFileUrl: result.CoreZipFileUrl,
		ArchBinFileUrl: result.ArchBinFileUrl,
	}

	return update, nil
}

type DownloadResult struct {
	Percent int
	Error   error
}

func DownloadFiles(coreFilesURL string, archBinURL string) chan DownloadResult {
	isDownloading := downloading.Load()

	if isDownloading {
		v := downloadCh.Load()
		return v.(chan DownloadResult)
	}

	resultCh := make(chan DownloadResult)
	downloadCh.Store(resultCh)
	downloading.Store(true)
	downloadPercent.Store(0)

	go func() {
		defer downloading.Store(false)
		defer downloadPercent.Store(0)
		defer close(resultCh)

		fileURLs := []string{coreFilesURL, archBinURL}
		totalPercent := len(fileURLs) * 100

		if err := sdkutils.FsEmptyDir(sdkutils.PathPluginUpdatesDir); err != nil {
			downloadError.Store(&err)
			resultCh <- DownloadResult{Error: err}
			return
		}

		for i, fileURL := range fileURLs {
			ch := downloadSystemFile(fileURL)

			for result := range ch {
				if result.Error != nil {
					log.Println("Error downloading system files:", result.Error)
					downloadError.Store(&result.Error)
					resultCh <- result
					return
				} else {
					progressPercent := result.Percent + (i * 100)
					percentVal := (float32(progressPercent) / float32(totalPercent)) * 100
					downloadPercent.Store(int32(percentVal))
					resultCh <- result
				}
			}
		}
	}()

	return resultCh
}

func downloadSystemFile(fileURL string) (resultCh chan DownloadResult) {
	resultCh = make(chan DownloadResult)
	downloadFilePath := filepath.Join(sdkutils.PathTmpDir, "system", "update", filepath.Base(fileURL))
	percentCh, errCh := sdkutils.DownloadFile(fileURL, downloadFilePath)

	go func() {
		defer close(resultCh)
		defer os.RemoveAll(downloadFilePath)

		for {
			result := DownloadResult{}
			select {
			case percent := <-percentCh:
				// do something with the percentage
				fmt.Println("Downloaded", percent, "%")
				result.Percent = percent
				resultCh <- result

			case err, ok := <-errCh:
				if !ok {
					return
				}

				if err != nil {
					fmt.Println("Error downloading software update", err)
					result.Error = ErrDownload
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

func IsDownloaded() bool {
	files := []string{
		"bin/flare",
		"core",
		"sdk",
	}

	for _, f := range files {
		if !sdkutils.FsExists(filepath.Join(sdkutils.PathPluginUpdatesDir, f)) {
			return false
		}
	}

	return true
}

func DownloadError() error {
	v := downloadError.Load()
	if v == nil {
		return nil
	}

	return *v
}

type InstallResult struct {
	Status string
	Error  string
}

func InstallUpdates() chan InstallResult {
	resultCh := make(chan InstallResult)
	go func() {
		// defer close(resultCh)
		// err := sdkutils.FsEmptyDir(SystemBackupPath)
		// if err != nil {
		// 	resultCh <- InstallResult{Error: err.Error()}
		// 	return
		// }
		// err = sdkutils.FsCopyDir(SystemUpdatePath, SystemBackupPath)
		// if err != nil {
		// 	resultCh <- InstallResult{Error: err.Error()}
		// 	return
		// }
		// err = sdkutils.FsCopyDir(SystemUpdatePath, sdkutils.PathStorageDir)
		// if err != nil {
		// 	resultCh <- InstallResult{Error: err.Error()}
		// 	return
		// }
		// resultCh <- InstallResult{Status: "success"}
	}()
	return resultCh
}
