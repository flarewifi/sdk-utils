package routes

import (
	"core/internal/plugins"
)

func PaymentRoutes(g *plugins.CoreGlobals) {

	// portalR := g.CoreAPI.HttpAPI.HttpRouter().PluginRouter()
	// vueR := g.CoreAPI.HttpAPI.VueRouter()

	// portalR.Group("/payments", func(subrouter sdkhttp.HttpRouterInstance) {
	// 	subrouter.Get("/options", controllers.PaymentOptionsCtrl(g)).Name("portal:payments:options")
	// })

	// vueR.RegisterPortalRoutes(sdkhttp.VuePortalRoute{
	// 	RouteName: "payments:customer:options",
	// 	RoutePath: "/payments/options",
	// 	Component: "payments/customer/PaymentOptions.vue",
	// })
}
