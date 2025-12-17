package routes

import (
	"net/http"

	"core/internal/api"
	"core/internal/web/controllers"
	"core/internal/web/middlewares"
	"core/internal/web/router"
	sdkapi "sdk/api"
)

func PortalRoutes(g *api.CoreGlobals) {

	coreAPI := g.CoreAPI
	rootR := router.RootRouter
	portalR := g.CoreAPI.HttpAPI.Router().PluginRouter()

	redirectToLanIpMw := middlewares.RedirectToLanIP(g.CoreAPI)
	httpRedirectMw := middlewares.HTTPRedirect()
	pendingPurchaseMw := middlewares.PendingPurchase(g.CoreAPI, g.Models)
	ensureDeviceMw := middlewares.EnsureDeviceRegistered(g.CoreAPI)

	portalSseCtrl := controllers.PortalSseHandler(g)
	portalRedirectCtrl := controllers.PortalRedirectCtrl(g)
	portalRegisterCtrl := controllers.PortalRegisterCtrl(g)
	portalRegisterAjaxCtrl := controllers.PortalRegisterAjaxCtrl(g)
	portalIndexPageCtrl := controllers.PortalIndexPage(g)

	// Root route redirects to /portal
	// Applies HTTPRedirect middleware to redirect HTTPS to HTTP
	rootR.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		h := redirectToLanIpMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			coreAPI.HttpAPI.Response().Redirect(w, r, "portal:redirector")
		}))
		h = httpRedirectMw(h)
		h.ServeHTTP(w, r)
	}).Methods("GET").Name("portal:root")

	portalR.Group("/register", func(regR sdkapi.IHttpRouterInstance) {
		regR.Use(redirectToLanIpMw)
		regR.Use(httpRedirectMw)
		regR.Group("/", func(subrouter sdkapi.IHttpRouterInstance) {

			subrouter.Get("/redirect", portalRedirectCtrl).
				Name("portal:redirector")

			// /portal/register - applies HTTPRedirect middleware
			subrouter.Get("/register", func(w http.ResponseWriter, r *http.Request) {
				h := http.HandlerFunc(portalRegisterCtrl)
				h.ServeHTTP(w, r)
			}).Name("portal:register")

			// /portal/register/ajax - applies HTTPRedirect middleware
			subrouter.Post("/register/ajax", func(w http.ResponseWriter, r *http.Request) {
				h := http.HandlerFunc(portalRegisterAjaxCtrl)
				h.ServeHTTP(w, r)
			}).Name("portal:register:ajax")
		})
	})

	portalR.Group("/portal", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Use(redirectToLanIpMw)
		subrouter.Use(httpRedirectMw)
		subrouter.Use(ensureDeviceMw)

		// /portal/index - applies plugin middlewares, HTTPRedirect, PortalDeviceCheck and PendingPurchase middlewares
		subrouter.Get("/index", func(w http.ResponseWriter, r *http.Request) {
			// Start with the base handler
			handler := http.Handler(http.HandlerFunc(portalIndexPageCtrl))

			// Apply core middlewares (inner to outer)
			handler = pendingPurchaseMw(handler)
			handler = httpRedirectMw(handler)

			// Collect portal middlewares from ALL plugins (not just core)
			var portalMws []func(http.Handler) http.Handler
			for _, plugin := range g.PluginMgr.Plugins() {
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
