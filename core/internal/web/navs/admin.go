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
					g.CoreAPI.Translate("label", "control"),
					g.CoreAPI.Translate("label", "panel"),
					g.CoreAPI.Translate("label", "status"),
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
					g.CoreAPI.Translate("label", "network"),
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
					g.CoreAPI.Translate("label", "firmware"),
					g.CoreAPI.Translate("label", "patch"),
					g.CoreAPI.Translate("label", "version"),
				},
				Order: 2000, // Second item in System category
				Icon:  "<i class='bi bi-cloud-arrow-down'></i>",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Database"),
				RouteName: "admin:database:index",
				Keywords: []string{
					g.CoreAPI.Translate("label", "Database"),
					g.CoreAPI.Translate("label", "Database Settings"),
					g.CoreAPI.Translate("label", "Reset Database"),
					g.CoreAPI.Translate("label", "backup"),
					g.CoreAPI.Translate("label", "restore"),
					g.CoreAPI.Translate("label", "sql"),
					g.CoreAPI.Translate("label", "storage"),
				},
				Order: 3000, // Third item in System category
				Icon:  "<i class='bi bi-database'></i>",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Admin User"),
				RouteName: "admin:user:index",
				Keywords: []string{
					g.CoreAPI.Translate("label", "Admin User"),
					g.CoreAPI.Translate("label", "Password"),
					g.CoreAPI.Translate("label", "Change Password"),
					g.CoreAPI.Translate("label", "Account"),
					g.CoreAPI.Translate("label", "security"),
					g.CoreAPI.Translate("label", "credentials"),
					g.CoreAPI.Translate("label", "login"),
				},
				Order: 4000, // After Database, before Logs
				Icon:  "<i class='bi bi-person-gear'></i>",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Logs"),
				RouteName: "admin:logs:index",
				Keywords: []string{
					g.CoreAPI.Translate("label", "Log"),
					g.CoreAPI.Translate("label", "Logs"),
					g.CoreAPI.Translate("label", "Audit"),
					g.CoreAPI.Translate("label", "debug"),
					g.CoreAPI.Translate("label", "trace"),
					g.CoreAPI.Translate("label", "events"),
					g.CoreAPI.Translate("label", "monitoring"),
				},
				Order: 8500, // Sit directly above Reboot, after plugin items
				Icon:  "<i class='bi bi-file-earmark-text'></i>",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Reboot"),
				RouteName: "admin:power:reboot",
				Keywords: []string{
					g.CoreAPI.Translate("label", "Reboot"),
					g.CoreAPI.Translate("label", "Restart"),
					g.CoreAPI.Translate("label", "restart"),
					g.CoreAPI.Translate("label", "service"),
					g.CoreAPI.Translate("label", "reload"),
					g.CoreAPI.Translate("label", "reset"),
					g.CoreAPI.Translate("label", "boot"),
				},
				Order: 9000, // Last items in System category
				Icon:  "<i class='bi bi-arrow-clockwise'></i>",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "Shutdown"),
				RouteName: "admin:power:shutdown",
				Keywords: []string{
					g.CoreAPI.Translate("label", "Shutdown"),
					g.CoreAPI.Translate("label", "Power Off"),
					g.CoreAPI.Translate("label", "Turn Off"),
					g.CoreAPI.Translate("label", "halt"),
					g.CoreAPI.Translate("label", "stop"),
					g.CoreAPI.Translate("label", "poweroff"),
					g.CoreAPI.Translate("label", "terminate"),
				},
				Order: 9100, // Very last item
				Icon:  "<i class='bi bi-power'></i>",
			},
		}

		themesNavs := []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategoryThemes,
				Label:     g.CoreAPI.Translate("label", "Admin Dashboard"),
				RouteName: "admin:themes:admin",
				Keywords: []string{
					g.CoreAPI.Translate("label", "Admin Theme"),
					g.CoreAPI.Translate("label", "Dashboard Theme"),
					g.CoreAPI.Translate("label", "Admin Style"),
					g.CoreAPI.Translate("label", "appearance"),
					g.CoreAPI.Translate("label", "config"),
					g.CoreAPI.Translate("label", "settings"),
					g.CoreAPI.Translate("label", "layout"),
				},
				Order: 4100,
				Icon:  "<i class='bi bi-layout-text-window-reverse'></i>",
			},
			{
				Category:  sdkapi.NavCategoryThemes,
				Label:     g.CoreAPI.Translate("label", "Captive Portal"),
				RouteName: "admin:themes:portal",
				Keywords: []string{
					g.CoreAPI.Translate("label", "Portal Theme"),
					g.CoreAPI.Translate("label", "Login Theme"),
					g.CoreAPI.Translate("label", "Captive Portal Style"),
					g.CoreAPI.Translate("label", "wifi"),
					g.CoreAPI.Translate("label", "splash"),
					g.CoreAPI.Translate("label", "access"),
					g.CoreAPI.Translate("label", "authentication"),
				},
				Order: 4200,
				Icon:  "<i class='bi bi-phone'></i>",
			},
		}

		adminNavs := append(quickAccessNavs, systemNavs...)
		adminNavs = append(adminNavs, themesNavs...)
		return adminNavs
	})
}
