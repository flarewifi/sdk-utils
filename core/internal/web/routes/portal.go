package routes

import (
	"core/internal/plugins"
	webutil "core/internal/utils/web"
	"core/internal/web/controllers"
	"core/internal/web/middlewares"
)

func PortalRoutes(g *plugins.CoreGlobals) {
	deviceMw := middlewares.DeviceMiddleware(g.Db, g.ClientRegister)
	rootR := webutil.RootRouter
	portalR := g.CoreAPI.HttpAPI.HttpRouter().PluginRouter()
	// pendingPurchaseMw := g.CoreAPI.HttpAPI.Middlewares().PendingPurchase()

	portalIndexCtrl := controllers.PortalIndexPage(g)
	portalSseCtrl := controllers.PortalSseHandler(g)
	// portalItemsCtrl := controllers.PortalItemsHandler(g)

	rootR.Handle("/", deviceMw(portalIndexCtrl)).Methods("GET").Name("portal:index")
	portalR.Get("/events", portalSseCtrl).Name("portal:sse")
	// portalR.Get("/nav/items", portalItemsCtrl, pendingPurchaseMw).Name("portal:navs:items")
}
