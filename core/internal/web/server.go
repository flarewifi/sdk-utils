package web

import (
	"log"
	"net/http"

	"core/internal/api"
	webutil "core/internal/utils/web"
	"core/internal/web/controllers"
	"core/internal/web/navs"
	"core/internal/web/routes"
)

func SetupBootRoutes(g *api.CoreGlobals) {
	routes.PluginAssets(g)
	routes.BootingAssets(g)
	routes.BootRoutes(g)
}

func SetupAppRoutes(g *api.CoreGlobals) {
	// GET "/boot/status" 200 OK
	webutil.RootRouter.HandleFunc(controllers.BootStatusURL, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	routes.PluginAssets(g)
	routes.GlobalAssets(g)
	routes.PortalRoutes(g)
	routes.AdminRoutes(g)
	routes.PaymentRoutes(g)

	navs.SetAdminNavs(g)

	webutil.RootRouter.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Warning: unknown route requested: ", r.URL.Path)
		http.Redirect(w, r, "/", http.StatusFound)
	})
}
