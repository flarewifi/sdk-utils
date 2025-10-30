package web

import (
	"log"
	"net/http"

	"core/internal/api"
	webutil "core/internal/utils/web"
	"core/internal/web/controllers"
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
	// GET "/boot/status" 200 OK
	webutil.RootRouter.HandleFunc(controllers.BootStatusURL, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	routes.AssetsRoutes(g)
	routes.PortalRoutes(g)
	routes.AdminRoutes(g)
	routes.PaymentRoutes(g)
	routes.FormRoutes(g)

	navs.SetAdminNavs(g)
	forms.RegisterForms(g)

	webutil.RootRouter.PathPrefix("/plugins").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<ul>"))
		for _, p := range g.PluginMgr.Plugins() {
			w.Write([]byte("<li>"))
			w.Write([]byte(p.Dir()))
			w.Write([]byte("</li>"))
		}
		w.Write([]byte("</ul>"))
	})

	webutil.RootRouter.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Warning: unknown route requested: ", r.URL.Path)
		http.Redirect(w, r, "/", http.StatusFound)
	})
}
