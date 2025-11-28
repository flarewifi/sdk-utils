package routes

import (
	"net/http"

	"core/internal/api"
	webutil "core/internal/utils/web"
	"core/internal/web/controllers"
	"core/internal/web/middlewares"
	sdkapi "sdk/api"
)

func PortalRoutes(g *api.CoreGlobals) {
	coreAPI := g.CoreAPI
	rootR := webutil.RootRouter
	portalR := g.CoreAPI.HttpAPI.Router().PluginRouter()
	redirectToLanIpMw := middlewares.RedirectToLanIP(g.CoreAPI)
	pendingPurchaseMw := g.CoreAPI.HttpAPI.Middlewares().PendingPurchase()

	portalSseCtrl := controllers.PortalSseHandler(g)
	portalRedirectCtrl := controllers.PortalRedirectCtrl(g)
	portalRegisterCtrl := controllers.PortalRegisterCtrl(g)
	portalRegisterAjaxCtrl := controllers.PortalRegisterAjaxCtrl(g)
	portalIndexPageCtrl := controllers.PortalIndexPage(g)

	// Root route redirects to /portal
	rootR.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		h := redirectToLanIpMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			coreAPI.HttpAPI.Response().Redirect(w, r, "portal:redirector")
		}))
		h.ServeHTTP(w, r)
	}).Methods("GET").Name("portal:root")

	// Portal subrouter using PluginRouter
	portalR.Group("/portal", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Use(redirectToLanIpMw)

		subrouter.Get("/redirect", portalRedirectCtrl).
			Name("portal:redirector")

		subrouter.Get("/register", portalRegisterCtrl).
			Name("portal:register")

		subrouter.Post("/register/ajax", portalRegisterAjaxCtrl).
			Name("portal:register:ajax")

		subrouter.Get("/index", func(w http.ResponseWriter, r *http.Request) {
			h := pendingPurchaseMw(http.HandlerFunc(portalIndexPageCtrl))
			h.ServeHTTP(w, r)
		}).Name("portal:index")

		// /portal/events route (SSE)
		subrouter.Get("/events", portalSseCtrl).Name("portal:sse")
	})
}
