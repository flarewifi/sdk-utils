package controllers

import (
	"encoding/json"
	"net/http"

	"core/internal/api"
	"core/internal/web/helpers"
	"core/resources/views/boot"

	sdkutils "github.com/flarewifi/sdk-utils"
)

const (
	BootURL         = "/boot"
	BootStatusURL   = "/boot/status"
	BootProgressURL = "/boot/progress"
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
		status = ctrl.g.CoreAPI.Translate("info", "The software is updating. This will take a few minutes, please wait")
	} else {
		status = ctrl.g.CoreAPI.Translate("info", "The software is booting, please wait")
	}

	page := boot.BootPage(&boot.BootPageData{
		StatusURL:   BootStatusURL,
		ProgressURL: BootProgressURL,
		API:         ctrl.g.CoreAPI,
		JsSrc:       jsSrc,
		CssSrc:      cssSrc,
		Status:      status,
	})

	if err := page.Render(r.Context(), w); err != nil {
		w.Write([]byte("\n\nTemplate Error:" + err.Error()))
	}
}

// BootProgress returns the live boot timeline as JSON for the booting page to
// poll. It is served only by the BootingRouter; once boot completes and the app
// router takes over, the page redirects home (driven by BootStatusURL), so this
// endpoint disappearing is harmless — the client tolerates a failed fetch.
func (ctrl *BootCtrl) BootProgress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	if err := json.NewEncoder(w).Encode(ctrl.g.BootProgress.Snapshot()); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (ctrl *BootCtrl) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAssetPath := helpers.IsAssetPath(r.URL.Path)

		if r.Method == http.MethodGet && !isAssetPath {
			if r.URL.Path != BootURL && r.URL.Path != BootStatusURL && r.URL.Path != BootProgressURL {
				http.Redirect(w, r, BootURL, http.StatusSeeOther)
				return
			}
		}
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
