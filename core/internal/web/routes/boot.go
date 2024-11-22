package routes

import (
	"log"
	"net/http"

	"core/internal/plugins"
	webutil "core/internal/utils/web"
	"core/internal/web/controllers"
	"core/internal/web/routes/urls"
)

func BootRoutes(g *plugins.CoreGlobals) {
	bootCtrl := controllers.NewBootCtrl(g, g.PluginMgr, g.CoreAPI)
	r := webutil.BootingRouter
	r.Use(bootCtrl.Middleware)
	r.HandleFunc(urls.BOOT_URL, bootCtrl.IndexPage).Methods("GET")
	r.HandleFunc(urls.BOOT_STATUS_URL, bootCtrl.SseHandler).Methods("GET")

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Location not found: ", r.URL.Path)
		http.Redirect(w, r, urls.BOOT_URL, http.StatusFound)
	})
}
