package controllers

import (
	"net/http"

	"core/internal/api"
	"core/internal/web/helpers"
	"core/resources/views/boot"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	BootURL       = "/boot"
	BootStatusURL = "/boot/status"
)

func NewBootCtrl(g *api.CoreGlobals) BootCtrl {
	return BootCtrl{g}
}

type BootCtrl struct {
	g *api.CoreGlobals
}

func (ctrl *BootCtrl) BootPage(w http.ResponseWriter, r *http.Request) {
	h := ctrl.g.CoreAPI.HttpAPI.Helpers().(*api.HttpHelpers)

	manifest := ctrl.g.CoreAPI.AssetsManifest
	jsSrcFile, _ := manifest.BootAssets.Scripts["boot.js"]
	cssSrcFile, _ := manifest.BootAssets.Styles["boot.css"]

	jsSrc := h.DistPath(jsSrcFile)
	cssSrc := h.DistPath(cssSrcFile)

	isUpdating := sdkutils.FsExists(sdkutils.PathIsUpdated) || sdkutils.FsExists(sdkutils.PathIsReverted)

	var status string
	if isUpdating {
		status = "The software is updating... This will take a few minutes, please wait..."
	} else {
		status = "The software is booting, please wait..."
	}

	page := boot.BootPage(&boot.BootPageData{
		StatusURL: BootStatusURL,
		API:       ctrl.g.CoreAPI,
		JsSrc:     jsSrc,
		CssSrc:    cssSrc,
		Status:    status,
	})

	if err := page.Render(r.Context(), w); err != nil {
		w.Write([]byte("\n\nTemplate Error:" + err.Error()))
	}
}

func (ctrl *BootCtrl) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAssetPath := helpers.IsAssetPath(r.URL.Path)

		if r.Method == http.MethodGet && !isAssetPath {
			if r.URL.Path != BootURL && r.URL.Path != BootStatusURL {
				http.Redirect(w, r, BootURL, http.StatusSeeOther)
				return
			}
		}
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
