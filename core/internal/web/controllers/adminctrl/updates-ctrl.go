package adminctrl

import (
	"core/internal/api"
	"core/internal/utils/updates"
	updatesview "core/resources/views/admin/updates"
	"errors"
	"log"
	"net/http"
	sdkapi "sdk/api"
	"sync/atomic"

	"github.com/Masterminds/semver/v3"
)

const (
	EventInstallProgress = "software-update-progress"
)

var (
	newUpdate atomic.Value
)

func CheckUpdatesPageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		isDownloading := updates.IsDownloading()
		isDownloaded := updates.IsDownloaded()
		if isDownloaded {
			res.Redirect(w, r, "system.updates.download.done")
			return
		}
		if isDownloading {
			res.Redirect(w, r, "system.updates.download")
			return
		}

		newUpdate.Store(&updates.CoreReleaseUpdate{HasUpdate: false})
		page := updatesview.SoftwareUpdatesPage(api, nil)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
		})
	}
}

func QuerySoftwareUpdatesCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		coreInfo := api.Info()

		checkUpdateErr := errors.New(g.CoreAPI.Translate("error", "check_updates_error"))
		currentVersion, err := semver.NewVersion(coreInfo.Version)
		if err != nil {
			log.Println("Error:", err)
			page := updatesview.SoftwareUpdatesPage(api, checkUpdateErr)
			page.Render(r.Context(), w)
			return
		}

		result, err := updates.CheckCoreReleaseUpdate(currentVersion)
		if err != nil {
			log.Println("Error:", err)
			page := updatesview.SoftwareUpdatesPage(api, checkUpdateErr)
			page.Render(r.Context(), w)
			return
		}

		newUpdate.Store(result)

		var update updatesview.SoftwareUpdate
		if result.HasUpdate {
			update = updatesview.SoftwareUpdate{
				HasUpdate:      true,
				NewVersion:     result.Version.String(),
				CurrentVersion: currentVersion.String(),
			}
		}

		page := updatesview.CheckForUpdatesPartial(api, update, nil)
		page.Render(r.Context(), w)
	}
}

func DownloadUpdatePageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		v := newUpdate.Load()
		update, ok := v.(*updates.CoreReleaseUpdate)
		if !ok {
			res.Redirect(w, r, "system.updates.check")
			return
		}

		if !update.HasUpdate {
			res.Redirect(w, r, "system.updates.check")
			return
		}

		isDownloading := updates.IsDownloading()
		isDownloaded := updates.IsDownloaded()

		var percent int32
		if isDownloading {
			percent = updates.DownloadPercent()
		}

		if isDownloaded {
			percent = 100
		}

		page := updatesview.DownloadUpdatePage(api, EventInstallProgress, int(percent), nil)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})

		if !isDownloaded && !isDownloading {
			// Initiate the process of downloading and installing of software updates
			go updates.DownloadFiles(update.CoreZipFileUrl, update.ArchBinFileUrl)
		}
	}
}

func DownloadStatusPartialCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		if updates.IsDownloaded() {
			api.HttpAPI.Response().Redirect(w, r, "system.updates.download.done")
			return
		}

		percent := updates.DownloadPercent()
		err := updates.DownloadError()
		page := updatesview.DownloadStatusPartial(api, int(percent), err)
		if err := page.Render(r.Context(), w); err != nil {
			api.LoggerAPI.Error(err.Error())
		}
	}
}

func DownloadDoneCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()
		if !updates.IsDownloaded() {
			res.Redirect(w, r, "system.updates.check")
			return
		}

		page := updatesview.DownloadDonePage(api)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
		})
	}
}
