package app

import (
	sdkapi "sdk/api"
)

const (
	RouteNameLogin   = "auth.login"
	RouteNameLogout  = "auth.logout"
	RoutePortalItems = "portal.items"
	RouteAdminNavs   = "admin.navs"
	RoutePayments    = "save.settings"
)

func SetupRoutes(api sdkapi.IPluginApi) {
	pluginRouter := api.Http().Router().PluginRouter()
	pluginRouter.Get("/sessions/summary", PortalSessionSyncHandler(api)).Name("sessions.summary")

	pluginRouter.Group("/sessions", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/summary", PortalSessionSyncHandler(api)).Name("sessions.summary")
		subrouter.Get("/sync", TriggerSessionSync(api)).Name("sessions.sync")
		subrouter.Get("/navs", PortalNavItemsHandler(api)).Name("portal.navs")
	})

}
