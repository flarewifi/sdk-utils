package app

import (
	sdkapi "sdk/api"

	"com.flarego.default-theme/app/controllers"
)

const (
	RouteNameLogin   = "auth.login"
	RouteNameLogout  = "auth.logout"
	RoutePortalItems = "portal.items"
	RouteAdminNavs   = "admin.navs"
	RoutePayments    = "save.settings"
)

func SetupRoutes(api sdkapi.IPluginApi) {
	// pluginRouter := api.Http().HttpRouter().PluginRouter()
	adminRouter := api.Http().Router().AdminRouter()
	// pluginRouter.Get("/test", controllers.IndexCtrl(api)).Name("index")
	adminRouter.Get("/test/{name}", controllers.TestCtrl(api)).Name("test")
	// pluginRouter.Group("/auth", func(subrouter sdkhttp.HttpRouterInstance) {
	// 	subrouter.Post("/login", controllers.LoginCtrl(api)).Name(RouteNameLogin)
	// 	subrouter.Post("/logout", controllers.LogoutCtrl(api)).Name(RouteNameLogout)
	// })

	// adminRouter := api.Http().HttpRouter().AdminRouter()
	// adminRouter.Get("/navs", controllers.GetAdminNavs(api)).Name(RouteAdminNavs)

}
