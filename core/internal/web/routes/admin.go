package routes

import (
	"core/internal/api"
	webutil "core/internal/utils/web"
	"core/internal/web/controllers"
	"core/internal/web/controllers/adminctrl"
	sdkapi "sdk/api"
)

func AdminRoutes(g *api.CoreGlobals) {
	authMw := g.CoreAPI.HttpAPI.Middlewares().AdminAuth()
	rootR := webutil.RootRouter
	adminR := g.CoreAPI.HttpAPI.Router().AdminRouter()

	adminIndexCtrl := controllers.AdminIndexPage(g)
	adminLoginCtrl := controllers.AdminLoginCtrl(g)
	adminAuthCtrl := controllers.AdminAuthenticateCtrl(g)
	adminSseCtrl := controllers.AdminSseHandler(g)

	rootR.Handle("/admin", authMw(adminIndexCtrl)).Methods("GET").Name("admin:index")
	// TODO: enable csrf protection
	rootR.Handle("/login", adminLoginCtrl).Methods("GET").Name("admin:login")
	rootR.Handle("/login", adminAuthCtrl).Methods("POST").Name("admin:authenticate")
	adminR.Get("/events", adminSseCtrl).Name("admin:sse")

	adminR.Group("/themes", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/index", adminctrl.GetAvailableThemes(g)).Name("admin:themes:index")
		subrouter.Post("/save", adminctrl.SaveThemeSettings(g)).Name("admin:themes:save")
	})

	adminR.Group("/logs", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/index", adminctrl.LogsIndex(g)).Name("admin:logs:index")
		subrouter.Post("/search", adminctrl.LogsPostSearch(g)).Name("admin:logs:search")
	})

	adminR.Group("/plugins", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/", adminctrl.PluginsIndexCtrl(g)).Name("admin.plugins.index")
		subrouter.Get("/install", adminctrl.PluginInstallCtrl(g)).Name("admin.plugins.install")
		subrouter.Post("/install/zip", adminctrl.PluginInstallFromZip(g)).Name("admin.plugins.install.zip")
		// TODO: Implement install from github
		subrouter.Post("/install/github", adminctrl.PluginInstallFromZip(g)).Name("admin.plugins.install.github")
		subrouter.Get("/uninstall/{pkg}", adminctrl.UninstallPluginCtrl(g)).Name("admin.plugins.uninstall")
	})

	// adminR.Group("/plugins", func(subrouter sdkhttp.HttpRouterInstance) {
	// 	subrouter.Get("/index", adminctrl.PluginsIndexCtrl(g)).
	// 		Name("admin:plugins:index")

	// 	subrouter.Group("/store", func(storeSubrouter sdkhttp.HttpRouterInstance) {
	// 		storeSubrouter.Get("/index", adminctrl.PluginsStoreIndexCtrl(g)).
	// 			Name("admin:plugins:store:index")

	// 		storeSubrouter.Get("/plugins/plugin", adminctrl.ViewPluginCtrl(g)).
	// 			Name("admin:plugins:store:plugin")
	// 	})

	// 	subrouter.Post("/install", adminctrl.PluginsInstallCtrl(g)).
	// 		Name("admin:plugins:install")

	// 	subrouter.Post("/uninstall", adminctrl.UninstallPluginCtrl(g)).
	// 		Name("admin:plugins:uninstall")

	// 	subrouter.Post("/update", adminctrl.UpdatePluginCtrl(g)).
	// 		Name("admin:plugins:update")

	// 	subrouter.Get("/checkupdates", adminctrl.CheckPluginUpdatesCtrl(g)).
	// 		Name("admin:plugins:checkupdates")
	// })

	// adminR.Group("/upload", func(subrouter sdkhttp.HttpRouterInstance) {
	// 	subrouter.Post("/file", adminctrl.UploadFileCtrl(g)).
	// 		Name("admin:upload:file")

	// TODO: for future use-case
	// subrouter.Post("/files", adminctrl.UploadFilesCtrl(g)).
	// Name("admin:upload:files")
	// })

	// g.CoreAPI.HttpAPI.VueRouter().RegisterAdminRoutes([]sdkhttp.VueAdminRoute{
	// 	{
	// 		RouteName: "theme-picker",
	// 		RoutePath: "/theme-picker",
	// 		Component: "admin/ThemePicker.vue",
	// 	},
	// 	{
	// 		RouteName: "log-viewer",
	// 		RoutePath: "/log-viewer",
	// 		Component: "admin/LogViewer.vue",
	// 	},
	// 	{
	// 		RouteName: "plugins-index",
	// 		RoutePath: "/plugins",
	// 		Component: "admin/plugins/Index.vue",
	// 	},
	// 	{
	// 		RouteName: "plugins-new",
	// 		RoutePath: "/plugins/new",
	// 		Component: "admin/plugins/NewInstall.vue",
	// 	},
	// 	{
	// 		RouteName: "plugins-store",
	// 		RoutePath: "/plugins/store",
	// 		Component: "admin/plugins/PluginsStore.vue",
	// 	},
	// 	{
	// 		RouteName: "plugin",
	// 		RoutePath: "/plugins/store/plugin",
	// 		Component: "admin/plugins/PluginDetail.vue",
	// 	},
	// 	{
	// 		RouteName: "core-updates",
	// 		RoutePath: "/system-updates",
	// 		Component: "admin/CoreUpdates.vue",
	// 	},
	// }...)
}
