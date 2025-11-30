package routes

import (
	"core/internal/api"
	webutil "core/internal/utils/web"
	"core/internal/web/controllers"
	"core/internal/web/controllers/adminctrl"
	"core/internal/web/middlewares"
	"net/http"
	sdkapi "sdk/api"
)

func AdminRoutes(g *api.CoreGlobals) {
	authMw := middlewares.AdminAuth(g.CoreAPI)
	trackNavMw := middlewares.TrackNav(g.Models)
	httpsRedirectMw := middlewares.HTTPSRedirect()
	rootR := webutil.RootRouter
	adminR := g.CoreAPI.HttpAPI.Router().AdminRouter()

	// Register HTTPS redirect and navigation tracking middleware to admin router
	adminR.Use(httpsRedirectMw)
	adminR.Use(trackNavMw)

	adminLoginCtrl := controllers.AdminLoginCtrl(g)
	adminSseCtrl := adminctrl.AdminSseHandler(g)

	webutil.AdminRouter.Handle("/", httpsRedirectMw(authMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.CoreAPI.HttpAPI.Response().Redirect(w, r, "admin:dashboard")
	})))).Methods("GET")

	adminR.Get("/dashboard", adminctrl.AdminIndexCtrl(g)).Name("admin:dashboard")

	// TODO: enable csrf protection
	rootR.Handle("/login", httpsRedirectMw(adminLoginCtrl)).Methods("GET").Name("admin:login")
	adminR.Get("/events", adminSseCtrl).Name(middlewares.RouteNameAdminSSE)

	adminR.Group("/system", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Group("/general", func(subrouter sdkapi.IHttpRouterInstance) {
			subrouter.Get("/index", adminctrl.GeneralSettingsIndexCtrl(g)).Name("admin:general:index")
			subrouter.Post("/save", adminctrl.GeneralSettingsSaveCtrl(g)).Name("admin:general:save")
		})
		subrouter.Group("/updates", func(subrouter sdkapi.IHttpRouterInstance) {
			subrouter.Get("/", adminctrl.CheckUpdatesPageCtrl(g)).Name("admin:updates:index")
			subrouter.Post("/query", adminctrl.QuerySoftwareUpdatesCtrl(g)).Name("admin:updates:query")
			subrouter.Get("/download", adminctrl.DownloadUpdatePageCtrl(g)).Name("admin:updates:download")
			subrouter.Post("/download/status", adminctrl.DownloadStatusPartialCtrl(g)).Name("admin:updates:download-status")
			subrouter.Get("/download/done", adminctrl.DownloadDoneCtrl(g)).Name("admin:updates:download-done")
			subrouter.Get("/sysupgrade", adminctrl.SysupgradePageCtrl(g)).Name("admin:updates:sysupgrade")
			subrouter.Post("/sysupgrade/upload", adminctrl.SysupgradeUploadCtrl(g)).Name("admin:updates:sysupgrade-upload")
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
		subrouter.Get("/reboot", adminctrl.RebootPageCtrl(g)).Name("admin:power:reboot")
		subrouter.Post("/reboot", adminctrl.RebootCtrl(g)).Name("admin:power:reboot-action")
		subrouter.Get("/shutdown", adminctrl.ShutdownPageCtrl(g)).Name("admin:power:shutdown")
		subrouter.Post("/shutdown", adminctrl.ShutdownCtrl(g)).Name("admin:power:shutdown-action")
	})
}
