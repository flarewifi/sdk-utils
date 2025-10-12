package adminctrl

import (
	"context"
	"core/internal/api"
	"core/internal/utils/updates"
	updatesview "core/resources/views/admin/updates"
	"errors"
	"log"
	"net/http"
	sdkapi "sdk/api"
	"strings"
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
		page := updatesview.SoftwareUpdatesPage(api)
		newUpdate.Store(&updates.CoreReleaseUpdate{HasUpdate: false})

		isDownloading := updates.IsDownloading()
		isDownloaded := updates.IsDownloaded()
		if isDownloading || isDownloaded {
			res.Redirect(w, r, "system.updates.install")
			return
		}

		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
		})
	}
}

func QuerySoftwareUpdatesCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()
		coreInfo := api.Info()

		checkUpdateErr := errors.New(g.CoreAPI.Translate("error", "check_updates_error"))
		currentVersion, err := semver.NewVersion(coreInfo.Version)
		if err != nil {
			log.Println("Error:", err)
			res.Error(w, r, checkUpdateErr, http.StatusInternalServerError)
			return
		}

		result, err := updates.CheckCoreReleaseUpdate(currentVersion)
		if err != nil {
			log.Println("Error:", err)
			res.Error(w, r, checkUpdateErr, http.StatusInternalServerError)
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

func InstallUpdatePageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		acct, err := api.HttpAPI.Auth().CurrentAcct(r)
		if err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, "system.updates.check")
			return
		}

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
			go func() {
				ch := updates.DownloadFiles(update.CoreZipFileUrl, update.ArchBinFileUrl)
				for result := range ch {
					partial := updatesview.DownloadStatusPartial(api, result.Percent, result.Error)
					var b strings.Builder
					if err := partial.Render(context.Background(), &b); err != nil {
						api.LoggerAPI.Error(err.Error())
						return
					}

					html := strings.ReplaceAll(b.String(), "\n", " ") // Strip new lines, refer to SSE spec
					acct.Emit(EventInstallProgress, []byte(html))

					if result.Error != nil {
						return
					}
				}
			}()
		}
	}
}

func InstallStatusCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		isDownloaded := updates.IsDownloaded()
		if isDownloaded {
			w.Header().Add("HX-Redirect", api.HttpAPI.Helpers().UrlForRoute(""))
		}

		downloadPercent := updates.DownloadPercent()
		downloadError := updates.DownloadError()

		partial := updatesview.DownloadStatusPartial(api, int(downloadPercent), downloadError)
		partial.Render(r.Context(), w)
	}
}
