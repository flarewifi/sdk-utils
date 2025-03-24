package adminctrl

import (
	"core/internal/api"
	"core/internal/utils/updates"
	updatesview "core/resources/views/admin/updates"
	"log"
	"net/http"
	sdkapi "sdk/api"
	"time"

	"github.com/Masterminds/semver/v3"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

func ShowUpdatesCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()
		coreInfo := api.Info()
		currentVersion, err := semver.NewVersion(coreInfo.Version)
		if err != nil {
			log.Println("Error:", err)
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		result, err := updates.CheckCoreReleaseUpdate(currentVersion)
		if err != nil {
			log.Println("Error:", err)
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		var update *updatesview.SoftwareUpdate
		if result.HasUpdate {
			update = &updatesview.SoftwareUpdate{
				Version: result.Version.String(),
			}
		}

		page := updatesview.ShowUpdates(api, update, nil)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
		})

	}
}

func QueryUpdatesCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()
		coreInfo := api.Info()
		currentVersion, err := semver.NewVersion(coreInfo.Version)
		if err != nil {
			log.Println("Error:", err)
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		result, err := updates.CheckCoreReleaseUpdate(currentVersion)
		if err != nil {
			log.Println("Error:", err)
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		var update *updatesview.SoftwareUpdate
		if result.HasUpdate {
			update = &updatesview.SoftwareUpdate{
				Version: result.Version.String(),
			}
		}

		time.Sleep(3 * time.Second)
		page := updatesview.QueryUpdate(api, update, nil)
		page.Render(r.Context(), w)
	}
}

func GetCurrentCoreVersionCtrl(g *api.CoreGlobals) http.HandlerFunc {
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

func DownloadUpdatesCtrl(g *api.CoreGlobals) http.HandlerFunc {
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
	downloader := sdkutils.NewDownloader(src, dest)
	err := downloader.Download()
	if err != nil {
		log.Println("Error:", err)
		return err
	}

	return nil
}

func UpdateCoreCtrl(g *api.CoreGlobals) http.HandlerFunc {
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
