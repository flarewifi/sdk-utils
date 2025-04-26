package routes

import (
	"core/internal/api"
	webutil "core/internal/utils/web"
	"core/internal/web/controllers"
	"core/internal/web/middlewares"
)

func PortalRoutes(g *api.CoreGlobals) {
	noCacheMw := middlewares.NoCache()
	deviceMw := middlewares.DeviceMiddleware(g.Db, g.ClientRegister)
	rootR := webutil.RootRouter
	portalR := g.CoreAPI.HttpAPI.Router().PluginRouter()
	pendingPurchaseMw := g.CoreAPI.HttpAPI.Middlewares().PendingPurchase()
	redirectToLanIpMw := middlewares.RedirectToLanIP(g.CoreAPI)
	checkDeviceStatusMw := middlewares.CheckDeviceStatus(g.CoreAPI)

	portalSseCtrl := controllers.PortalSseHandler(g)
	portalIndexCtrl := controllers.PortalIndexPage(g)

	// add middlewares to portal controller
	portalIndexCtrl = noCacheMw(portalIndexCtrl)
	portalIndexCtrl = deviceMw(portalIndexCtrl)
	portalIndexCtrl = redirectToLanIpMw(portalIndexCtrl)
	portalIndexCtrl = pendingPurchaseMw(portalIndexCtrl)
	portalIndexCtrl = checkDeviceStatusMw(portalIndexCtrl)

	rootR.Handle("/", portalIndexCtrl).Methods("GET").Name("portal.index")
	portalR.Get("/events", portalSseCtrl).Name("portal.sse")
}
