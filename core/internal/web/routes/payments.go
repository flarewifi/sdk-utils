package routes

import (
	"core/internal/api"
	"core/internal/web/controllers"
	sdkapi "sdk/api"
)

func PaymentRoutes(g *api.CoreGlobals) {

	portalR := g.CoreAPI.HttpAPI.Router().HttpRouter(nil)
	paymentsCtrl := controllers.PaymentOptionsCtrl(g)
	optionsListCtrl := controllers.PaymentOptionsListCtrl(g)
	cancelPurchaseCtrl := controllers.CancelPurchaseCtrl(g)

	portalR.Group("/payments", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/options", paymentsCtrl).Name("payments:options")
		subrouter.Post("/cancel", cancelPurchaseCtrl).Name("portal:payments:cancel")
		subrouter.Get("/options-list", optionsListCtrl).Name("payments:options:list")
	})
}
