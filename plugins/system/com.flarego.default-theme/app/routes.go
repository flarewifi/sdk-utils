package app

import (
	sdkplugin "sdk/api/plugin"

	"com.flarego.default-theme/app/controllers"
)

const (
	RouteNameLogin   = "auth.login"
	RouteNameLogout  = "auth.logout"
	RoutePortalItems = "portal.items"
	RouteAdminNavs   = "admin.navs"
	RoutePayments    = "save.settings"
)

func SetupRoutes(api sdkplugin.IPluginApi) {
	// pluginRouter := api.Http().HttpRouter().PluginRouter()
	adminRouter := api.Http().HttpRouter().AdminRouter()
	// pluginRouter.Get("/test", controllers.IndexCtrl(api)).Name("index")
	adminRouter.Get("/test", controllers.TestCtrl(api)).Name("test")
	// pluginRouter.Group("/auth", func(subrouter sdkhttp.HttpRouterInstance) {
	// 	subrouter.Post("/login", controllers.LoginCtrl(api)).Name(RouteNameLogin)
	// 	subrouter.Post("/logout", controllers.LogoutCtrl(api)).Name(RouteNameLogout)
	// })

	// adminRouter := api.Http().HttpRouter().AdminRouter()
	// adminRouter.Get("/navs", controllers.GetAdminNavs(api)).Name(RouteAdminNavs)

}
