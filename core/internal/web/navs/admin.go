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
				Label:     g.CoreAPI.Translate("label", "Dashboard"),
				RouteName: "admin:dashboard",
				Keywords:  []string{"dashboard"},
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "General"),
				RouteName: "admin:general:index",
				Keywords:  []string{"settings", "general", "language", "currency", "version", "machine", "id"},
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Updates"),
				RouteName: "admin:updates:index",
				Keywords:  []string{"update", "updates", "upgrade", "upgrades", "software"},
			},
		}

		// Append plugin navs
		systemNavs = append(systemNavs, GetAdminPluginNavs(g)...)

		powerNavs := []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Logs"),
				RouteName: "admin:logs:index",
				Keywords:  []string{"log", "logs", "audit", "audits"},
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Reboot"),
				RouteName: "admin:power:reboot",
				Keywords:  []string{"power", "reboot", "restart"},
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Shutdown"),
				RouteName: "admin:power:shutdown",
				Keywords:  []string{"power", "shutdown", "off"},
			},
		}

		systemNavs = append(systemNavs, powerNavs...)

		themesNavs := []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategoryThemes,
				Label:     g.CoreAPI.Translate("label", "Select Theme"),
				RouteName: "admin:themes:index",
				Keywords:  []string{"theme", "themes", "style", "portal", "admin"},
			},
		}

		adminNavs := append(systemNavs, themesNavs...)
		return adminNavs
	})

	GetAdminPluginNavs(g)
}
