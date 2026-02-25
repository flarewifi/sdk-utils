package routes

import (
	"core/internal/api"
	"core/internal/web/controllers"
	"core/internal/web/controllers/adminctrl"
	"core/internal/web/middlewares"
	"core/internal/web/router"
	"net/http"
	sdkapi "sdk/api"
)

func AdminRoutes(g *api.CoreGlobals) {
	authMw := middlewares.AdminAuth(g.CoreAPI)
	httpsRedirectMw := middlewares.HTTPSRedirect()
	rootR := router.RootRouter
	adminR := g.CoreAPI.HttpAPI.Router().AdminRouter()

	// Register HTTPS redirect middleware to admin router
	adminR.Use(httpsRedirectMw)

	adminLoginCtrl := controllers.AdminLoginCtrl(g)
	adminSseCtrl := adminctrl.AdminSseHandler(g)

	router.AdminRouter.Handle("/", httpsRedirectMw(authMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.CoreAPI.HttpAPI.Response().Redirect(w, r, "admin:dashboard")
	})))).Methods("GET")

	adminR.Get("/dashboard", adminctrl.AdminIndexCtrl(g)).Name("admin:dashboard")

	// TODO: enable csrf protection
	rootR.Handle("/login", httpsRedirectMw(adminLoginCtrl)).Methods("GET").Name("admin:login")
	adminR.Get("/events", adminSseCtrl).Name(api.RouteNameAdminSSE)

	adminR.Group("/system", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Group("/general", func(subrouter sdkapi.IHttpRouterInstance) {
			subrouter.Get("/index", adminctrl.GeneralSettingsIndexCtrl(g)).Name("admin:general:index")
			subrouter.Post("/save", adminctrl.GeneralSettingsSaveCtrl(g)).Name("admin:general:save")
			subrouter.Get("/system-resources", adminctrl.GeneralSettingsSystemResourcesCtrl(g)).Name("admin:general:system-resources")
		})
		subrouter.Group("/updates", func(subrouter sdkapi.IHttpRouterInstance) {
			subrouter.Get("/", adminctrl.CheckUpdatesPageCtrl(g)).Name("admin:updates:index")
			subrouter.Post("/query", adminctrl.QuerySoftwareUpdatesCtrl(g)).Name("admin:updates:query")
			subrouter.Get("/download", adminctrl.DownloadUpdatePageCtrl(g)).Name("admin:updates:download")
			subrouter.Post("/download/status", adminctrl.DownloadStatusPartialCtrl(g)).Name("admin:updates:download-status")
			subrouter.Get("/download/done", adminctrl.DownloadDoneCtrl(g)).Name("admin:updates:download-done")
			subrouter.Post("/sysupgrade/upload", adminctrl.SysupgradeUploadCtrl(g)).Name("admin:updates:sysupgrade-upload")
			subrouter.Get("/sysupgrade/success", adminctrl.SysupgradeSuccessPageCtrl(g)).Name("admin:updates:sysupgrade-success")
			subrouter.Get("/sysupgrade/progress", adminctrl.SysupgradeProgressPageCtrl(g)).Name("admin:updates:sysupgrade-progress")
			subrouter.Post("/sysupgrade/delete", adminctrl.SysupgradeDeleteCtrl(g)).Name("admin:updates:sysupgrade-delete")
		})
		subrouter.Group("/database", func(subrouter sdkapi.IHttpRouterInstance) {
			subrouter.Get("/index", adminctrl.DatabaseSettingsIndexCtrl(g)).Name("admin:database:index")
			subrouter.Post("/reset", adminctrl.DatabaseResetCtrl(g)).Name("admin:database:reset")
		})
		subrouter.Group("/user", func(subrouter sdkapi.IHttpRouterInstance) {
			subrouter.Get("/index", adminctrl.AdminUserIndexCtrl(g)).Name("admin:user:index")
			subrouter.Post("/change-password", adminctrl.AdminUserChangePasswordCtrl(g)).Name("admin:user:change-password")
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
