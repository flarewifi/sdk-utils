package adminctrl

import (
	"core/internal/plugins"
	"log"
	"net/http"

	sdkdownloader "github.com/flarehotspot/go-utils/downloader"
)

func FetchUpdatesCtrl(g *plugins.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.VueResponse()

		// latestCoreRelease, err := updates.FetchLatestCoreRelease()
		// if err != nil {
		// 	log.Println("Error fetching latest core release:", err)
		// 	res.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		// res.Json(w, latestCoreRelease, http.StatusOK)
	}
}

func GetCurrentCoreVersionCtrl(g *plugins.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.VueResponse()

		// coreVersion, err := updates.GetCurrentCoreVersion()
		// if err != nil {
		// 	log.Println("Error:", err)
		// 	res.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		// res.Json(w, coreVersion, http.StatusOK)
	}
}

func DownloadUpdatesCtrl(g *plugins.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.VueResponse()

		// var reqPayload updates.CoreReleaseUpdate
		// err := json.NewDecoder(r.Body).Decode(&reqPayload)
		// if err != nil {
		// 	res.Error(w, err.Error(), http.StatusBadRequest)
		// 	log.Println("Error reading the request body:", err)
		// 	return
		// }

		// stringedVersion := sdksemver.StringifyVersion(reqPayload.Version)
		// updatesPath := filepath.Join(sdkpaths.CacheDir, "updates", "core", stringedVersion)
		// coreFilesPath := filepath.Join(updatesPath, "core-files")
		// archBinFilesPath := filepath.Join(updatesPath, "arch-bin-files")

		// // download core files
		// err = downloadFile(reqPayload.CoreZipFileUrl, coreFilesPath)
		// if err != nil {
		// 	res.Error(w, err.Error(), http.StatusBadRequest)
		// 	log.Println("Error downloading core files:", err)
		// 	return
		// }
		// // download arch bin files
		// err = downloadFile(reqPayload.ArchBinFileUrl, archBinFilesPath)
		// if err != nil {
		// 	log.Println("Error downloading arch bin files:", err)
		// 	res.Error(w, err.Error(), http.StatusBadRequest)
		// 	return
		// }

		// // TODO: verify downloaded files by checking its checksum

		// // return downloaded local file paths
		// localUpdateFiles := updates.UpdateFiles{
		// 	LocalCoreFilesPath:    coreFilesPath,
		// 	LocalArchBinFilesPath: archBinFilesPath,
		// }

		// res.Json(w, localUpdateFiles, http.StatusOK)
	}
}

func downloadFile(src string, dest string) error {
	downloader := sdkdownloader.NewDownloader(src, dest)
	err := downloader.Download()
	if err != nil {
		log.Println("Error:", err)
		return err
	}

	return nil
}

func UpdateCoreCtrl(g *plugins.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.VueResponse()

		// var reqPayload updates.UpdateFiles
		// err := json.NewDecoder(r.Body).Decode(&reqPayload)
		// if err != nil {
		// 	res.Error(w, err.Error(), http.StatusBadRequest)
		// 	log.Println("Error reading the request body:", err)
		// 	return
		// }

		// if err := updates.UpdateCore(reqPayload); err != nil {
		// 	log.Println("Error:", err)
		// 	res.Error(w, err.Error(), http.StatusBadRequest)
		// 	return
		// }

		// res.Json(w, "Update in progress..", http.StatusOK)
	}
}
