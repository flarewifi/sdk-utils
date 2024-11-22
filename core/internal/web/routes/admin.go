package routes

import (
	"core/internal/plugins"
	webutil "core/internal/utils/web"
	"core/internal/web/controllers"
	"core/internal/web/controllers/adminctrl"
	sdkhttp "sdk/api/http"
)

func AdminRoutes(g *plugins.CoreGlobals) {
	authMw := g.CoreAPI.HttpAPI.Middlewares().AdminAuth()
	rootR := webutil.RootRouter
	adminR := g.CoreAPI.HttpAPI.HttpRouter().AdminRouter()

	adminIndexCtrl := controllers.AdminIndexPage(g)
	adminLoginCtrl := controllers.AdminLoginCtrl(g)
	adminAuthCtrl := controllers.AdminAuthenticateCtrl(g)
	adminSseCtrl := controllers.AdminSseHandler(g)
	adminFormsCtrl := adminctrl.NewFormsCtrl(g)

	rootR.Handle("/admin", authMw(adminIndexCtrl)).Methods("GET").Name("admin:index")
	// TODO: enable csrf protection
	rootR.Handle("/login", adminLoginCtrl).Methods("GET").Name("admin:login")
	rootR.Handle("/login", adminAuthCtrl).Methods("POST").Name("admin:authenticate")

	adminR.Group("/forms", func(subrouter sdkhttp.IHttpRouterInstance) {
		subrouter.Post("/save", adminFormsCtrl.SaveForm).Queries("pkg", "{pkg}", "name", "{name}").Name("admin:forms:save")
	})

	adminR.Get("/events", adminSseCtrl).Name("admin:sse")

	adminR.Group("/themes", func(subrouter sdkhttp.IHttpRouterInstance) {
		subrouter.Get("/index", adminctrl.GetAvailableThemes(g)).Name("admin:themes:index")
		subrouter.Get("/save", adminctrl.SaveThemeSettings(g)).Name("admin:themes:save")
	})

	// adminR.Group("/core", func(subrouter sdkhttp.HttpRouterInstance) {
	// 	subrouter.Get("/fetch", adminctrl.FetchUpdatesCtrl(g)).Name("admin:core:fetch")
	// 	subrouter.Get("/current", adminctrl.GetCurrentCoreVersionCtrl(g)).Name("admin:core:current")
	// 	subrouter.Post("/download", adminctrl.DownloadUpdatesCtrl(g)).Name("admin:core:download")
	// 	subrouter.Post("/update", adminctrl.UpdateCoreCtrl(g)).Name("admin:core:update")
	// })

	// adminR.Group("/logs", func(subrouter sdkhttp.HttpRouterInstance) {
	// 	subrouter.Get("/index", adminctrl.GetLogs(g)).Name("admin:logs:index")
	// 	subrouter.Post("/clear", adminctrl.ClearLogs(g)).Name("admin:logs:clear")
	// })

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

	// g.CoreAPI.HttpAPI.VueRouter().AdminNavsFunc(func(acct sdkacct.Account) []sdkhttp.VueAdminNav {
	// 	return []sdkhttp.VueAdminNav{
	// 		{
	// 			Category:  sdkhttp.NavCategoryThemes,
	// 			Label:     "Select Theme",
	// 			RouteName: "theme-picker",
	// 		},
	// 		{
	// 			Category:  sdkhttp.NavCategorySystem,
	// 			Label:     "View Logs",
	// 			RouteName: "log-viewer",
	// 		},
	// 		{
	// 			Category:  sdkhttp.NavCategorySystem,
	// 			Label:     "Manage Plugins",
	// 			RouteName: "plugins-index",
	// 		},
	// 		{
	// 			Category:  sdkhttp.NavCategorySystem,
	// 			Label:     "System Updates",
	// 			RouteName: "core-updates",
	// 		},
	// 	}
	// })
}
