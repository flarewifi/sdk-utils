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

	portalSseCtrl := controllers.PortalSseHandler(g)
	portalRedirectCtrl := controllers.PortalRedirectCtrl(g)
	portalRegisterCtrl := controllers.PortalRegisterCtrl(g)
	portalRegisterAjaxCtrl := controllers.PortalRegisterAjaxCtrl(g)
	portalIndexPageCtrl := controllers.PortalIndexPage(g)

	// Root route redirects to /portal
	// Applies HTTPRedirect middleware to redirect HTTPS to HTTP
	rootR.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		h := httpRedirectMw(redirectToLanIpMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			coreAPI.HttpAPI.Response().Redirect(w, r, "portal:redirector")
		})))
		h.ServeHTTP(w, r)
	}).Methods("GET").Name("portal:root")

	// Portal subrouter using PluginRouter
	portalR.Group("/portal", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Use(redirectToLanIpMw)

		subrouter.Get("/redirect", portalRedirectCtrl).
			Name("portal:redirector")

		// /portal/register - applies HTTPRedirect middleware
		subrouter.Get("/register", func(w http.ResponseWriter, r *http.Request) {
			h := httpRedirectMw(http.HandlerFunc(portalRegisterCtrl))
			h.ServeHTTP(w, r)
		}).Name("portal:register")

		// /portal/register/ajax - applies HTTPRedirect middleware
		subrouter.Post("/register/ajax", func(w http.ResponseWriter, r *http.Request) {
			h := httpRedirectMw(http.HandlerFunc(portalRegisterAjaxCtrl))
			h.ServeHTTP(w, r)
		}).Name("portal:register:ajax")

		// /portal/index - applies HTTPRedirect and PendingPurchase middlewares
		subrouter.Get("/index", func(w http.ResponseWriter, r *http.Request) {
			h := httpRedirectMw(pendingPurchaseMw(http.HandlerFunc(portalIndexPageCtrl)))
			h.ServeHTTP(w, r)
		}).Name("portal:index")

		// /portal/events route (SSE)
		subrouter.Get("/events", portalSseCtrl).Name("portal:sse")
	})
}
