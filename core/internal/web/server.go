package web

import (
	"log"
	"net/http"

	"core/internal/api"
	webutil "core/internal/utils/web"
	forms "core/internal/web/forms"
	"core/internal/web/navs"
	"core/internal/web/routes"
)

func SetupBootRoutes(g *api.CoreGlobals) {
	routes.AssetsRoutes(g)
	routes.CoreAssets(g)
	routes.BootRoutes(g)
}

func SetupAppRoutes(g *api.CoreGlobals) {
	routes.AssetsRoutes(g)
	routes.PortalRoutes(g)
	routes.AdminRoutes(g)
	routes.PaymentRoutes(g)
	routes.FormRoutes(g)

	navs.SetAdminNavs(g)
	forms.RegisterForms(g)

	webutil.RootRouter.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Warning: unknown route requested: ", r.URL.Path)
		http.Redirect(w, r, "/", http.StatusFound)
	})
}
