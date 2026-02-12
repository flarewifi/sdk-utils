package navs

import (
	"core/internal/api"
	"net/http"
	sdkapi "sdk/api"
)

// SetAdminNavs registers the core system's admin navigation items.
// Navigation items are sorted by the Order field within each category.
// Order guidelines:
//   - 1000-3000: Core system items (General, Updates, Database)
//   - 4000-5000: Plugin settings and features (default: 5000)
//   - 6000-8000: Less frequently used items
//   - 9000+: Items that should appear last (Reboot, Shutdown)
func SetAdminNavs(g *api.CoreGlobals) {
	coreNavs := g.CoreAPI.HttpAPI.Navs()

	coreNavs.AdminNavsFactory(func(r *http.Request) []sdkapi.AdminNavItemOpt {
		quickAccessNavs := []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategoryQuickAccess,
				Label:     g.CoreAPI.Translate("label", "Dashboard"),
				RouteName: "admin:dashboard",
				Keywords:  []string{"dashboard", "home", "main", "overview"},
				Order:     1000,
			},
		}

		systemNavs := []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "General"),
				RouteName: "admin:general:index",
				Keywords: []string{
					g.CoreAPI.Translate("label", "General Settings"),
					g.CoreAPI.Translate("label", "Device Info"),
					g.CoreAPI.Translate("label", "Language"),
					g.CoreAPI.Translate("label", "Currency"),
					g.CoreAPI.Translate("label", "Software Version"),
					g.CoreAPI.Translate("label", "Machine ID"),
				},
				Order: 1000, // First item in System category
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Updates"),
				RouteName: "admin:updates:index",
				Keywords:  []string{"update", "updates", "upgrade", "upgrades", "software"},
				Order:     2000, // Second item in System category
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Database"),
				RouteName: "admin:database:index",
				Keywords: []string{
					g.CoreAPI.Translate("label", "Database"),
					g.CoreAPI.Translate("label", "Database Settings"),
					g.CoreAPI.Translate("label", "Reset Database"),
					"sqlite", "postgresql", "postgres",
				},
				Order: 3000, // Third item in System category
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Admin User"),
				RouteName: "admin:user:index",
				Keywords:  []string{"admin", "user", "password", "account", "profile"},
				Order:     4000, // After Database (3000), before Logs (5000)
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Logs"),
				RouteName: "admin:logs:index",
				Keywords:  []string{"log", "logs", "audit", "audits"},
				Order:     5000, // Default position (after plugin items with Order < 5000)
			},
		}

		// Append plugin navs
		systemNavs = append(systemNavs, GetAdminPluginNavs(g)...)

		themesNavs := []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategoryThemes,
				Label:     g.CoreAPI.Translate("label", "Select Theme"),
				RouteName: "admin:themes:index",
				Keywords:  []string{"theme", "themes", "style", "portal", "admin"},
			},
		}

		// Power controls should appear last in System category to prevent accidental clicks.
		// Using very high Order values (9998, 9999) ensures they appear after all other items.
		powerNavs := []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Reboot"),
				RouteName: "admin:power:reboot",
				Keywords:  []string{"power", "reboot", "restart"},
				Order:     9998, // Second to last in System category
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Shutdown"),
				RouteName: "admin:power:shutdown",
				Keywords:  []string{"power", "shutdown", "off"},
				Order:     9999, // Last item in System category
			},
		}

		adminNavs := append(quickAccessNavs, systemNavs...)
		adminNavs = append(adminNavs, themesNavs...)
		adminNavs = append(adminNavs, powerNavs...)
		return adminNavs
	})

	GetAdminPluginNavs(g)
}
