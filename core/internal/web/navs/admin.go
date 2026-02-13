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
				Keywords: []string{
					g.CoreAPI.Translate("label", "Dashboard"),
					g.CoreAPI.Translate("label", "Home"),
					g.CoreAPI.Translate("label", "Main"),
					g.CoreAPI.Translate("label", "Overview"),
				},
				Order: 1000,
				Icon:  "<i class='bi bi-columns-gap'></i>",
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
				Icon:  "<i class='bi bi-gear'></i>",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Updates"),
				RouteName: "admin:updates:index",
				Keywords: []string{
					g.CoreAPI.Translate("label", "Update"),
					g.CoreAPI.Translate("label", "Updates"),
					g.CoreAPI.Translate("label", "Upgrade"),
					g.CoreAPI.Translate("label", "Software"),
				},
				Order: 2000, // Second item in System category
				Icon:  "<i class='bi bi-arrow-clockwise'></i>",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Database"),
				RouteName: "admin:database:index",
				Keywords: []string{
					g.CoreAPI.Translate("label", "Database"),
					g.CoreAPI.Translate("label", "Database Settings"),
					g.CoreAPI.Translate("label", "Reset Database"),
				},
				Order: 3000, // Third item in System category
				Icon:  "<i class='bi bi-database'></i>",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Logs"),
				RouteName: "admin:logs:index",
				Keywords: []string{
					g.CoreAPI.Translate("label", "Log"),
					g.CoreAPI.Translate("label", "Logs"),
					g.CoreAPI.Translate("label", "Audit"),
				},
				Order: 5000, // Default position (after plugin items with Order < 5000)
				Icon:  "<i class='bi bi-file-earmark-text'></i>",
			},
		}

		// Append plugin navs
		systemNavs = append(systemNavs, GetAdminPluginNavs(g)...)

		themesNavs := []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategoryThemes,
				Label:     g.CoreAPI.Translate("label", "Select Theme"),
				RouteName: "admin:themes:index",
				Keywords: []string{
					g.CoreAPI.Translate("label", "Theme"),
					g.CoreAPI.Translate("label", "Themes"),
					g.CoreAPI.Translate("label", "Style"),
					g.CoreAPI.Translate("label", "Portal"),
					g.CoreAPI.Translate("label", "Admin"),
				},
				Icon: "<i class='bi bi-palette'></i>",
			},
		}

		adminNavs := append(quickAccessNavs, systemNavs...)
		adminNavs = append(adminNavs, themesNavs...)
		return adminNavs
	})

	GetAdminPluginNavs(g)
}
