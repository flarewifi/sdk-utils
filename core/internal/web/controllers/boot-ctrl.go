package controllers

import (
	"log"
	"net/http"
	"path"

	"core/internal/api"
	sse "core/internal/utils/sse"
	"core/internal/web/helpers"
	"core/internal/web/routes/urls"
	"core/resources/views/boot"
)

func NewBootCtrl(g *api.CoreGlobals, pmgr *api.PluginsMgr, api *api.PluginApi) BootCtrl {
	return BootCtrl{g.BootProgress, pmgr, api}
}

type BootCtrl struct {
	bp   *api.BootProgress
	pmgr *api.PluginsMgr
	api  *api.PluginApi
}

func (ctrl *BootCtrl) IndexPage(w http.ResponseWriter, r *http.Request) {
	jsSrc := ctrl.api.HttpAPI.Helpers().ResourcePath(path.Join("assets", "booting", "js", "booting.js"))
	cssSrc := ctrl.api.HttpAPI.Helpers().ResourcePath(path.Join("assets", "booting", "css", "style.css"))

	// globals := plugins.ReadGlobalAssetsManifest()
	// jsSrc := ctrl.api.CoreAPI.Http().Helpers().ResourcePath(path.Join("assets", "dist", globals.BootingJsFile))
	// cssSrc := ctrl.api.CoreAPI.Http().Helpers().ResourcePath(path.Join("assets", "dist", globals.BootingCssFile))

	page := boot.BootPage(&boot.BootPageData{
		SseURL: urls.BOOT_STATUS_URL,
		JsSrc:  jsSrc,
		CssSrc: cssSrc,
	})

	w.Header().Set("Content-Type", "text/html")
	if err := page.Render(r.Context(), w); err != nil {
		w.Write([]byte("\n\nTemplate Error:" + err.Error()))
	}
}

func (ctrl *BootCtrl) SseHandler(w http.ResponseWriter, r *http.Request) {
	s, err := sse.NewSocket(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctrl.bp.AddSocket(s)
	s.Listen()
}

func (ctrl *BootCtrl) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		done := ctrl.bp.IsDone()

		isAssetPath := helpers.IsAssetPath(r.URL.Path)
		log.Printf("Is asset path: %s => %v", r.URL.Path, isAssetPath)

		if r.Method == "GET" && !isAssetPath {
			if !done && r.URL.Path != urls.BOOT_URL && r.URL.Path != urls.BOOT_STATUS_URL {
				http.Redirect(w, r, urls.BOOT_URL, http.StatusSeeOther)
				return
			}

			if done && r.URL.Path == urls.BOOT_URL {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
