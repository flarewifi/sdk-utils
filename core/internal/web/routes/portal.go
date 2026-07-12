package routes

import (
	"net/http"

	"core/internal/api"
	"core/internal/web/controllers"
	"core/internal/web/middlewares"
	"core/internal/web/router"
	"core/utils/plugins"
	sdkapi "sdk/api"
)

func PortalRoutes(g *api.CoreGlobals) {

	rootR := router.RootRouter
	portalR := g.CoreAPI.HttpAPI.Router().HttpRouter(nil)

	// Scheme/host/destination normalization — HTTPS, the portal-domain funnel,
	// and the unmanaged→/admin gate — is handled ONCE, globally, by
	// middlewares.ForceHTTPS on RootRouter, which runs for every route matched
	// here. Portal routes therefore carry NO per-route funnel wrappers (the 404
	// fallback in server.go is the one place that still wraps manually, because
	// mux Use middlewares never run on NotFoundHandler).
	pendingPurchaseMw := middlewares.PendingPurchase(g.CoreAPI, g.Models)
	ensureDeviceMw := middlewares.EnsureDeviceRegistered(g.CoreAPI)

	portalSseCtrl := controllers.PortalSseHandler(g)
	portalRootCtrl := controllers.PortalRootCtrl(g)
	captiveApiCtrl := controllers.CaptiveApiCtrl(g)
	portalRedirectCtrl := controllers.PortalRedirectCtrl(g)
	portalRegisterCtrl := controllers.PortalRegisterCtrl(g)
	portalRegisterAjaxCtrl := controllers.PortalRegisterAjaxCtrl(g)
	portalIndexPageCtrl := controllers.PortalIndexPage(g)

	// Root route renders a simple HTML page with inline JavaScript that advances
	// the portal flow. ForceHTTPS has already normalized entry on
	// localhost/LAN-IP/HTTP onto the portal hostname over HTTPS.
	rootR.Handle("/", http.HandlerFunc(portalRootCtrl)).Methods("GET").Name("portal:root")

	// RFC 8908 Captive Portal API (advertised via RFC 8910 DHCP option 114).
	// Registered on the root router so the OS reaches it at the advertised portal
	// hostname (captive.<domain>) without the redirect-to-LAN-IP middleware.
	rootR.HandleFunc("/api/captive", captiveApiCtrl).Methods("GET").Name("portal:captive-api")

	portalR.Group("/", func(regR sdkapi.IHttpRouterInstance) {
		// /register/redirect
		regR.Get("/register/redirect", portalRedirectCtrl).
			Name("portal:redirector")

		// /register
		regR.Get("/register", portalRegisterCtrl).
			Name("portal:register")

		// /register/ajax
		regR.Post("/register/ajax", portalRegisterAjaxCtrl).
			Name("portal:register:ajax")
	})

	// Core fallback theme HTMX endpoints — used by the built-in portal theme
	// (index.templ / status_nav.templ) when the configured theme plugin is absent.
	portalR.Group("/portal", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Use(ensureDeviceMw)
		subrouter.Get("/status-nav", controllers.PortalStatusNavCtrl(g)).Name("portal:status-nav")
		subrouter.Get("/sessions/summary", controllers.PortalSessionSummaryCtrl(g)).Name("portal:sessions:summary")
		subrouter.Get("/navs", controllers.PortalNavsCtrl(g)).Name("portal:navs")
	})

	portalR.Group("/portal", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Use(ensureDeviceMw)

		// /portal/index — wraps the page in PendingPurchase plus every
		// plugin-registered portal middleware (UseForPortal), outermost first.
		subrouter.Get("/index", func(w http.ResponseWriter, r *http.Request) {
			// Start with the base handler
			handler := http.Handler(http.HandlerFunc(portalIndexPageCtrl))

			// Apply core middlewares (inner to outer)
			handler = pendingPurchaseMw(handler)

			// Collect portal middlewares from ALL plugins (not just core), skipping
			// any plugin that is currently blocked/disabled/update-skipped/queued
			// for uninstall -- otherwise a plugin withheld from every other route
			// (see middlewares.PluginValidityCheck) would still gate live captive-
			// portal traffic through its UseForPortal middleware.
			var portalMws []func(http.Handler) http.Handler
			for _, plugin := range g.PluginMgr.PluginApis() {
				if plugins.IsInvalid(plugin.Info().Package) {
					continue
				}
				pluginMws := plugin.Http().Router().(*api.HttpRouterApi).GetPortalMiddlewares()
				portalMws = append(portalMws, pluginMws...)
			}

			// Apply plugin-registered middlewares (outer to inner)
			for i := len(portalMws) - 1; i >= 0; i-- {
				handler = portalMws[i](handler)
			}

			handler.ServeHTTP(w, r)
		}).Name("portal:index")

		// /portal/events route (SSE)
		subrouter.Get("/events", portalSseCtrl).Name("portal:sse")
	})
}
