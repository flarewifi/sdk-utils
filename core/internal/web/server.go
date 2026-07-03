package web

import (
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
	// Force HTTPS globally, before any other middleware, so the admin pages and
	// the captive portal always run over TLS. RootRouter backs both the HTTP and
	// HTTPS listeners; this redirects the HTTP side (admin -> same host, portal ->
	// portal domain) while keeping port 80 open to intercept captive-portal probes.
	router.RootRouter.Use(middlewares.ForceHTTPS())

	// Apply global activation check middleware (before any routes)
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
	routes.ForgotPasswordRoutes(g)
	routes.PaymentRoutes(g)
	routes.WifiEventRoutes(g)

	navs.SetAdminNavs(g)

	redirectToLanIpMw := middlewares.RedirectToLanIP(g.CoreAPI)
	redirectToPortalMw := middlewares.RedirectToPortalDomain()

	router.RootRouter.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
