package routes

import (
	"core/internal/api"
	webutil "core/internal/utils/web"
	"core/internal/web/controllers"
	"core/internal/web/routes/urls"
	"log"
	"net/http"
)

func BootRoutes(g *api.CoreGlobals) {
	bootCtrl := controllers.NewBootCtrl(g, g.PluginMgr, g.CoreAPI)

	r := webutil.BootingRouter
	r.Use(bootCtrl.Middleware)
	r.HandleFunc(urls.BOOT_URL, bootCtrl.BootPage).Methods(http.MethodGet)
	r.HandleFunc(urls.BOOT_STATUS_URL, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusExpectationFailed)
	}).Methods(http.MethodGet)

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Location not found: ", r.URL.Path)
		log.Println("Redirecting to boot page: ", urls.BOOT_URL)
		http.Redirect(w, r, urls.BOOT_URL, http.StatusFound)
	})
}
