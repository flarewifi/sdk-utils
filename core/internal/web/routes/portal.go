package routes

import (
	"core/internal/api"
	webutil "core/internal/utils/web"
	"core/internal/web/controllers"
	"core/internal/web/middlewares"
)

func PortalRoutes(g *api.CoreGlobals) {
	deviceMw := middlewares.DeviceMiddleware(g.Db, g.ClientRegister)
	rootR := webutil.RootRouter
	portalR := g.CoreAPI.HttpAPI.Router().PluginRouter()
	pendingPurchaseMw := g.CoreAPI.HttpAPI.Middlewares().PendingPurchase()

	portalIndexCtrl := controllers.PortalIndexPage(g)
	portalSseCtrl := controllers.PortalSseHandler(g)

	rootR.Handle("/", pendingPurchaseMw(deviceMw(portalIndexCtrl))).Methods("GET").Name("portal.index")
	portalR.Get("/events", portalSseCtrl).Name("portal:sse")
}
