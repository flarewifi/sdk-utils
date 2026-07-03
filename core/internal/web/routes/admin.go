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
	rootR := router.RootRouter
	adminR := g.CoreAPI.HttpAPI.Router().AdminRouter(nil)

	// HTTPS is enforced globally by middlewares.ForceHTTPS (applied on RootRouter
	// in SetupAppRoutes), so no per-route HTTPS redirect is needed here.

	adminLoginCtrl := controllers.AdminLoginCtrl(g)
	adminSseCtrl := adminctrl.AdminSseHandler(g)

	router.AdminRouter.Handle("/", authMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.CoreAPI.HttpAPI.Response().Redirect(w, r, "admin:dashboard")
	}))).Methods("GET")

	adminR.Get("/dashboard", adminctrl.AdminIndexCtrl(g)).Name("admin:dashboard")

	// TODO: enable csrf protection
	rootR.Handle("/login", adminLoginCtrl).Methods("GET").Name("admin:login")
	rootR.Handle("/login", controllers.AdminAuthenticateCtrl(g)).Methods("POST").Name("auth:login")

	// Register auth:login on the core static plugin router so UrlForRoute("auth:login")
	// resolves correctly when the fallback core theme is active (com.flarego.core#static#auth:login).
	g.CoreAPI.HttpAPI.Router().HttpRouter(&sdkapi.HttpRouterOpts{Static: true}).Post("/login", controllers.AdminAuthenticateCtrl(g)).Name("auth:login")
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
			subrouter.Post("/download/continue", adminctrl.DownloadContinueCtrl(g)).Name("admin:updates:download-continue")
			subrouter.Post("/download/cancel", adminctrl.DownloadCancelCtrl(g)).Name("admin:updates:download-cancel")
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
			subrouter.Post("/clear-history", adminctrl.AdminUserClearHistoryCtrl(g)).Name("admin:user:clear-history")
		})
	})

	// Core fallback theme routes — must exist so layout.templ / notifications.templ
	// can call UrlForRoute even when the configured theme plugin is not loaded.
	adminR.Post("/logout", controllers.AdminLogoutCtrl(g).ServeHTTP).Name("admin:auth:logout")

	adminR.Group("/notifications", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/count", adminctrl.NotificationsBellCountCtrl(g)).Name("admin:notifications:count")
		subrouter.Get("/list", adminctrl.NotificationsListCtrl(g)).Name("admin:notifications:list")
		subrouter.Get("/show/{id}", adminctrl.ShowNotificationContentCtrl(g)).Name("admin:notifications:show")
		subrouter.Post("/update/{id}", adminctrl.UpdateNotificationCtrl(g)).Name("admin:notifications:update")
		subrouter.Post("/clear-all", adminctrl.ClearAllNotificationsCtrl(g)).Name("admin:notifications:clear-all")
	})

	adminR.Group("/network", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Group("/interfaces", func(subrouter sdkapi.IHttpRouterInstance) {
			subrouter.Get("/index", adminctrl.InterfacesIndexCtrl(g)).Name("admin:interfaces:index")
			subrouter.Post("/save", adminctrl.InterfacesSaveCtrl(g)).Name("admin:interfaces:save")
			subrouter.Post("/apply", adminctrl.InterfacesApplyCtrl(g)).Name("admin:interfaces:apply")
		})
	})

	adminR.Group("/themes", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/admin", adminctrl.AdminThemesPage(g)).Name("admin:themes:admin")
		subrouter.Get("/portal", adminctrl.PortalThemesPage(g)).Name("admin:themes:portal")
		subrouter.Post("/save", adminctrl.SaveThemeSettings(g)).Name("admin:themes:save")
	})

	adminR.Group("/logs", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/index", adminctrl.LogsIndex(g)).Name("admin:logs:index")
		subrouter.Post("/search", adminctrl.LogsPostSearch(g)).Name("admin:logs:search")
		subrouter.Get("/stream", adminctrl.LogsStream(g)).Name("admin:logs:stream")
	})

	adminR.Group("/devices", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/diag", adminctrl.DeviceDiagCtrl(g)).Name("admin:devices:diag")
		subrouter.Get("/logs", adminctrl.DeviceLogsCtrl(g)).Name("admin:devices:logs")
		subrouter.Post("/clear-history", adminctrl.DeviceClearHistoryCtrl(g)).Name("admin:devices:clear-history")
	})

	adminR.Group("/power", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/reboot", adminctrl.RebootPageCtrl(g)).Name("admin:power:reboot")
		subrouter.Post("/reboot", adminctrl.RebootCtrl(g)).Name("admin:power:reboot-action")
		subrouter.Get("/shutdown", adminctrl.ShutdownPageCtrl(g)).Name("admin:power:shutdown")
		subrouter.Post("/shutdown", adminctrl.ShutdownCtrl(g)).Name("admin:power:shutdown-action")
	})
}
