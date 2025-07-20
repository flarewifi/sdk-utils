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
				Label:     g.CoreAPI.Translate("label", "timezone"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "logs"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "diagnostics"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "software_updates"),
				RouteName: "system.updates.check",
			},

			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "backup_restore"),
				RouteName: "system.updates.check",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "account_settings"),
				RouteName: "system.updates.check",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "themes"),
				RouteName: "admin:themes:index",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "plugins"),
				RouteName: "admin.plugins.index",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "logs"),
				RouteName: "admin:logs:index",
			},

			{
				Category:  sdkapi.NavCategoryWifi,
				Label:     g.CoreAPI.Translate("label", "rates_settings"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategoryWifi,
				Label:     g.CoreAPI.Translate("label", "session_settings"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategoryWifi,
				Label:     g.CoreAPI.Translate("label", "wifi_customers"),
				RouteName: "#",
			},

			{
				Category:  sdkapi.NavCategoryPayments,
				Label:     g.CoreAPI.Translate("label", "main_vendo_gpio_settings"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategoryPayments,
				Label:     g.CoreAPI.Translate("label", "wireless_subvbendo_premium"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategoryPayments,
				Label:     g.CoreAPI.Translate("label", "wired_subvbendo_premium"),
				RouteName: "#",
			},

			{
				Category:  sdkapi.NavCategoryUsers,
				Label:     g.CoreAPI.Translate("label", "devices"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategoryUsers,
				Label:     g.CoreAPI.Translate("label", "sessions"),
				RouteName: "#",
			},

			{
				Category:  sdkapi.NavCategoryThemes,
				Label:     g.CoreAPI.Translate("label", "captive_portal"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategoryThemes,
				Label:     g.CoreAPI.Translate("label", "admin"),
				RouteName: "#",
			},

			{
				Category:  sdkapi.NavCategoryPlugins,
				Label:     g.CoreAPI.Translate("label", "installed_plugins"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategoryPlugins,
				Label:     g.CoreAPI.Translate("label", "available_plugins"),
				RouteName: "#",
			},

			{
				Category:  sdkapi.NavCategoryNetwork,
				Label:     g.CoreAPI.Translate("label", "wan"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategoryNetwork,
				Label:     g.CoreAPI.Translate("label", "lan"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategoryNetwork,
				Label:     g.CoreAPI.Translate("label", "wireless"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategoryNetwork,
				Label:     g.CoreAPI.Translate("label", "bridges"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategoryNetwork,
				Label:     g.CoreAPI.Translate("label", "bandwidth_limit"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategoryNetwork,
				Label:     g.CoreAPI.Translate("label", "security"),
				RouteName: "#",
			},
			{
				Category:  sdkapi.NavCategoryNetwork,
				Label:     g.CoreAPI.Translate("label", "content_control"),
				RouteName: "#",
			},
		}
	})
}
