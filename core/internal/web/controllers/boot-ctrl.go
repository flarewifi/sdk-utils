package controllers

import (
	"net/http"
	"path"

	"core/internal/api"
	"core/internal/utils/plugins"
	"core/internal/web/helpers"
	"core/internal/web/routes/urls"
	"core/resources/views/boot"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func NewBootCtrl(g *api.CoreGlobals, pmgr *api.PluginsMgr, api *api.PluginApi) BootCtrl {
	return BootCtrl{pmgr, api}
}

type BootCtrl struct {
	pmgr *api.PluginsMgr
	api  *api.PluginApi
}

func (ctrl *BootCtrl) BootPage(w http.ResponseWriter, r *http.Request) {
	globals := plugins.ReadGlobalAssetsManifest()
	jsSrc := ctrl.api.CoreAPI.Http().Helpers().ResourcePath(path.Join("assets", "dist", globals.BootingJsFile))
	cssSrc := ctrl.api.CoreAPI.Http().Helpers().ResourcePath(path.Join("assets", "dist", globals.BootingCssFile))
	isUpdating := sdkutils.FsExists(sdkutils.PathIsUpdated) || sdkutils.FsExists(sdkutils.PathIsReverted)

	var status string
	if isUpdating {
		status = "The software is updating... This will take a few minutes, please wait..."
	} else {
		status = "The software is booting, please wait..."
	}

	page := boot.BootPage(&boot.BootPageData{
		API:    ctrl.api,
		JsSrc:  jsSrc,
		CssSrc: cssSrc,
		Status: status,
	})

	if err := page.Render(r.Context(), w); err != nil {
		w.Write([]byte("\n\nTemplate Error:" + err.Error()))
	}
}

func (ctrl *BootCtrl) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAssetPath := helpers.IsAssetPath(r.URL.Path)

		if r.Method == http.MethodGet && !isAssetPath {
			if r.URL.Path != urls.BOOT_URL && r.URL.Path != urls.BOOT_STATUS_URL {
				http.Redirect(w, r, urls.BOOT_URL, http.StatusSeeOther)
				return
			}
		}
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
