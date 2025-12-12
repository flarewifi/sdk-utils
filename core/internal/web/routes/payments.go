package routes

import (
	"core/internal/api"
	"core/internal/web/controllers"
	sdkapi "sdk/api"
)

func PaymentRoutes(g *api.CoreGlobals) {

	portalR := g.CoreAPI.HttpAPI.Router().PluginRouter()
	paymentsCtrl := controllers.PaymentOptionsCtrl(g)
	cancelPurchaseCtrl := controllers.CancelPurchaseCtrl(g)

	portalR.Group("/payments", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/options", paymentsCtrl).Name("payments:options")
		subrouter.Post("/cancel", cancelPurchaseCtrl).Name("portal:payments:cancel")
	})
}
