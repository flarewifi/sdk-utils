package navs

import (
	"core/internal/api"
	"net/http"
	sdkapi "sdk/api"
)

func SetAdminNavs(g *api.CoreGlobals) {
	coreNavs := g.CoreAPI.HttpAPI.Navs()

	coreNavs.AdminNavsFactory(func(r *http.Request) []sdkapi.AdminNavItemOpt {
		systemNavs := []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "dashboard"),
				RouteName: "admin:dashboard",
				Keywords:  []string{"dashboard"},
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "general"),
				RouteName: "admin:general:index",
				Keywords:  []string{"settings", "general", "language", "currency", "version", "machine", "id"},
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "updates"),
				RouteName: "system.updates.check",
				Keywords:  []string{"update", "updates", "upgrade", "upgrades", "software"},
			},
		}

		// Append plugin navs
		systemNavs = append(systemNavs, GetAdminPluginNavs(g)...)

		powerNavs := []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "logs"),
				RouteName: "admin:logs:index",
				Keywords:  []string{"log", "logs", "audit", "audits"},
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "reboot"),
				RouteName: "admin.power.reboot",
				Keywords:  []string{"power", "reboot", "restart"},
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "shutdown"),
				RouteName: "admin.power.shutdown",
				Keywords:  []string{"power", "shutdown", "off"},
			},
		}

		systemNavs = append(systemNavs, powerNavs...)

		themesNavs := []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategoryThemes,
				Label:     g.CoreAPI.Translate("label", "select_theme"),
				RouteName: "admin:themes:index",
				Keywords:  []string{"theme", "themes", "style", "portal", "admin"},
			},
		}

		adminNavs := append(systemNavs, themesNavs...)
		return adminNavs
	})

	GetAdminPluginNavs(g)
}
