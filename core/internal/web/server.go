package web

import (
	"log"
	"net/http"
	"strings"

	"core/internal/api"
	"core/internal/web/controllers"
	"core/internal/web/middlewares"
	"core/internal/web/navs"
	"core/internal/web/router"
	"core/internal/web/routes"
)

func SetupBootRoutes(g *api.CoreGlobals) {
	routes.PluginAssets(g)
	routes.BootingAssets(g)
	routes.BootRoutes(g)
}

func SetupAppRoutes(g *api.CoreGlobals) {
	// Apply global activation check middleware FIRST (before any routes)
	// This only runs after booting completes (when RootRouter is active)
	activationCheckMw := middlewares.ActivationCheck()
	router.RootRouter.Use(activationCheckMw)

	router.RootRouter.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// GET "/boot/status" 200 OK
	router.RootRouter.HandleFunc(controllers.BootStatusURL, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Register activation route
	routes.ActivationRoutes(g)

	routes.PluginAssets(g)
	routes.GlobalAssets(g)
	routes.PortalRoutes(g)
	routes.AdminRoutes(g)
	routes.PaymentRoutes(g)
	routes.WifiEventRoutes(g)

	navs.SetAdminNavs(g)

	redirectToLanIpMw := middlewares.RedirectToLanIP(g.CoreAPI)
	redirectToPortalMw := middlewares.RedirectToPortalDomain()

	router.RootRouter.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Warning: unknown route requested: ", r.URL.Path)

		// Admin 404s stay on the device host (LAN IP / localhost).
		if strings.HasPrefix(r.URL.Path, "/admin") {
			h := redirectToLanIpMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/admin", http.StatusFound)
			}))
			h.ServeHTTP(w, r)
			return
		}

		// Portal 404s — including OS captive-detection probes that hit arbitrary
		// URLs — are funneled to the portal hostname over HTTPS, not the bare LAN IP.
		h := redirectToPortalMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/", http.StatusFound)
		}))
		h.ServeHTTP(w, r)
	})
}
