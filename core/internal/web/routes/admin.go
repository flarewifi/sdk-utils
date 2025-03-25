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

	adminR.Group("/system", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Group("/updates", func(subrouter sdkapi.IHttpRouterInstance) {
			subrouter.Get("/check", adminctrl.ShowUpdatesCtrl(g)).Name("system.updates.check")
			subrouter.Post("/query", adminctrl.CheckUpdatesCtrl(g)).Name("system.updates.query")
		})
	})

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
		subrouter.Get("/install", adminctrl.PluginInstallIndexCtrl(g)).Name("admin.plugins.install")
		subrouter.Post("/install/zip", adminctrl.PluginInstallFromZipCtrl(g)).Name("admin.plugins.install.zip")
		subrouter.Post("/install/github", adminctrl.PluginsInstallFromGitCtrl(g)).Name("admin.plugins.install.github")
		subrouter.Post("/uninstall/{pkg}", adminctrl.UninstallPluginCtrl(g)).Name("admin.plugins.uninstall")
		subrouter.Get("/checkupdates/{pkg}", adminctrl.CheckPluginUpdatesCtrl(g)).Name("admin.plugins.checkupdates")
		subrouter.Get("/getupdate/{pkg}/{tag}", adminctrl.DownloadPluginUpdatesCtrl(g)).Name("admin.plugins.getupdate")
	})
}
