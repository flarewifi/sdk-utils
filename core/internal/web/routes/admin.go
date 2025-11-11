package routes

import (
	"core/internal/api"
	webutil "core/internal/utils/web"
	"core/internal/web/controllers"
	"core/internal/web/controllers/adminctrl"
	"net/http"
	sdkapi "sdk/api"
)

func AdminRoutes(g *api.CoreGlobals) {
	authMw := g.CoreAPI.HttpAPI.Middlewares().AdminAuth()
	rootR := webutil.RootRouter
	adminR := g.CoreAPI.HttpAPI.Router().AdminRouter()

	adminLoginCtrl := controllers.AdminLoginCtrl(g)
	adminAuthCtrl := controllers.AdminAuthenticateCtrl(g)
	adminSseCtrl := adminctrl.AdminSseHandler(g)

	rootR.Handle("/admin", authMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.CoreAPI.HttpAPI.Response().Redirect(w, r, "admin:dashboard")
	}))).Methods("GET")

	adminR.Get("/dashboard", adminctrl.AdminDashboardCtrl(g)).Name("admin:dashboard")

	// TODO: enable csrf protection
	rootR.Handle("/login", adminLoginCtrl).Methods("GET").Name("admin:login")
	rootR.Handle("/login", adminAuthCtrl).Methods("POST").Name("admin:authenticate")
	adminR.Get("/events", adminSseCtrl).Name("admin:sse")

	adminR.Group("/system", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Group("/general", func(subrouter sdkapi.IHttpRouterInstance) {
			subrouter.Get("/index", adminctrl.GeneralSettingsIndexCtrl(g)).Name("admin:general:index")
			subrouter.Post("/save", adminctrl.GeneralSettingsSaveCtrl(g)).Name("admin:general:save")
		})
		subrouter.Group("/updates", func(subrouter sdkapi.IHttpRouterInstance) {
			subrouter.Get("/", adminctrl.CheckUpdatesPageCtrl(g)).Name("system.updates.check")
			subrouter.Post("/query", adminctrl.QuerySoftwareUpdatesCtrl(g)).Name("system.updates.query")
			subrouter.Get("/download", adminctrl.DownloadUpdatePageCtrl(g)).Name("system.updates.download")
			subrouter.Post("/download/status", adminctrl.DownloadStatusPartialCtrl(g)).Name("system.updates.download.status")
			subrouter.Get("/download/done", adminctrl.DownloadDoneCtrl(g)).Name("system.updates.download.done")
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

	AdminPluginRoutes(g)

	adminR.Group("/power", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/reboot", adminctrl.RebootPageCtrl(g)).Name("admin.power.reboot")
		subrouter.Post("/reboot", adminctrl.RebootCtrl(g)).Name("admin.power.reboot.action")
		subrouter.Get("/shutdown", adminctrl.ShutdownPageCtrl(g)).Name("admin.power.shutdown")
		subrouter.Post("/shutdown", adminctrl.ShutdownCtrl(g)).Name("admin.power.shutdown.action")
	})
}
