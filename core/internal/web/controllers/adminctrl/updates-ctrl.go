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
)

func ShowUpdatesCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()
		page := updatesview.ShowUpdates(api)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
		})

	}
}

func CheckUpdatesCtrl(g *api.CoreGlobals) http.HandlerFunc {
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
				NewVersion:     result.Version.String(),
				CurrentVersion: currentVersion.String(),
			}
		}

		time.Sleep(3 * time.Second)
		page := updatesview.CheckForUpdate(api, update, nil)
		page.Render(r.Context(), w)
	}
}
