package app

import (
	sdkapi "sdk/api"

	"com.flarego.default-theme/app/handlers"
)

const (
	RouteNameAuthenticate = "auth:login"
	RouteNameLogout       = "admin:auth:logout"
)

func SetupRoutes(api sdkapi.IPluginApi) {
	adminR := api.Http().Router().AdminRouter()
	pluginR := api.Http().Router().PluginRouter()

	pluginR.Group("/sessions", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/summary", PortalSessionSyncHandler(api)).Name("portal:sessions:summary")
		subrouter.Get("/navs", PortalNavItemsHandler(api)).Name("portal:navs")
		subrouter.Get("/status-nav", PortalStatusNavHandler(api)).Name("portal:status-nav")
		subrouter.Get("/wif-rates", PortalWifiRatesCtrl(api)).Name("portal:wifi-rates")
	})

	adminR.Group("/dashboard", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/sales-summary", handlers.DashboardSalesCtrl(api)).Name("admin:dashboard:sales")
		subrouter.Get("/active-data", handlers.DashboardActiveDataCtrl(api)).Name("admin:dashboard:active-data")
		subrouter.Get("/internet-status", handlers.DashboardInternetStatusCtrl(api)).Name("admin:dashboard:internet-status")
		subrouter.Get("/revenue-chart", handlers.DashboardRevenueChartCtrl(api)).Name("admin:dashboard:revenue-chart")
	})

	adminR.Group("/system", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Group("/resource", func(subrouter sdkapi.IHttpRouterInstance) {
			subrouter.Get("/", handlers.SystemResourceCtrl(api)).Name("admin:system:resource")
		})
	})

	adminR.Group("/notifications", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/list", handlers.NotificationsListCtrl(api)).Name("admin:notifications:list")
		subrouter.Post("/update/{id}", handlers.UpdateNotificationCtrl(api)).Name("admin:notifications:update")
		subrouter.Post("/clear-all", handlers.ClearAllNotificationsCtrl(api)).Name("admin:notifications:clear-all")
		subrouter.Get("/count", handlers.NotificationsBellCountCtrl(api)).Name("admin:notifications:count")
		subrouter.Get("/show/{id}", handlers.ShowNotificationContentCtrl(api)).Name("admin:notifications:show")
	})

	adminR.Group("/themes", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/admin", handlers.AdminThemesPageCtrl(api)).Name("admin:themes:admin")
		subrouter.Get("/portal", handlers.PortalThemesPageCtrl(api)).Name("admin:themes:portal")
	})

	pluginR.Post("/login", handlers.AdminAuthenticateCtrl(api)).Name(RouteNameAuthenticate)
	adminR.Post("/logout", handlers.LogoutCtrl(api)).Name(RouteNameLogout)
}
