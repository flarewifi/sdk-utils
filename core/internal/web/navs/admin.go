package navs

import (
	"core/internal/api"
	"net/http"
	sdkapi "sdk/api"
)

func SetAdminNavs(g *api.CoreGlobals) {
	coreNavs := g.CoreAPI.HttpAPI.Navs()

	coreNavs.AdminNavsFactory(func(r *http.Request) []sdkapi.AdminNavItemOpt {
		return []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "themes"),
				RouteName: "admin:themes:index",
				Keywords:  []string{"theme", "themes", "style"},
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "plugins"),
				RouteName: "admin.plugins.index",
				Keywords:  []string{"plugin", "plugins", "extension", "extensions"},
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "logs"),
				RouteName: "admin:logs:index",
				Keywords:  []string{"log", "logs", "audit", "audits"},
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "software_updates"),
				RouteName: "system.updates.check",
				Keywords:  []string{"update", "updates", "upgrade", "upgrades", "software"},
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "reboot"),
				RouteName: "system.power.reboot",
				Keywords:  []string{"power", "reboot", "restart"},
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "shutdown"),
				RouteName: "system.power.shutdown",
				Keywords:  []string{"power", "shutdown", "off"},
			},
		}
	})
}
