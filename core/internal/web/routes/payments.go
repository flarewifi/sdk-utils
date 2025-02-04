package routes

import (
	"core/internal/plugins"
	"core/internal/web/controllers"
	sdkapi "sdk/api"
)

func PaymentRoutes(g *plugins.CoreGlobals) {

	portalR := g.CoreAPI.HttpAPI.HttpRouter().PluginRouter()
	paymentsCtrl := controllers.PaymentOptionsCtrl(g)

	portalR.Group("/payments", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/options", paymentsCtrl).Name("payments:options")
	})
}
