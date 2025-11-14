//go:build !mono

package routes

import (
	"core/internal/api"
	"core/internal/web/controllers/adminctrl"
	sdkapi "sdk/api"
)

func AdminPluginRoutes(g *api.CoreGlobals) {
	adminR := g.CoreAPI.HttpAPI.Router().AdminRouter()

	adminR.Group("/plugins", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/", adminctrl.PluginsIndexCtrl(g)).Name("admin:plugins:index")
		subrouter.Get("/install", adminctrl.PluginInstallIndexCtrl(g)).Name("admin:plugins:install")
		subrouter.Post("/install/zip", adminctrl.PluginInstallFromZipCtrl(g)).Name("admin:plugins:install-zip")
		subrouter.Post("/install/github", adminctrl.PluginsInstallFromGitCtrl(g)).Name("admin:plugins:install-github")
		subrouter.Post("/uninstall/{pkg}", adminctrl.UninstallPluginCtrl(g)).Name("admin:plugins:uninstall")
		subrouter.Get("/checkupdates/{pkg}", adminctrl.CheckPluginUpdatesCtrl(g)).Name("admin:plugins:check-updates")
		subrouter.Get("/getupdate/{pkg}/{tag}", adminctrl.DownloadPluginUpdatesCtrl(g)).Name("admin:plugins:get-update")
		subrouter.Get("/status", adminctrl.CheckPluginStatusCtrl(g)).Name("admin:plugins:status")
	})
}
