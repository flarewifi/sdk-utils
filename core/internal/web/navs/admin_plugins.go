//go:build !mono

package navs

import (
	"core/internal/api"
	sdkapi "sdk/api"
)

func GetAdminPluginNavs(g *api.CoreGlobals) []sdkapi.AdminNavItemOpt {
	return []sdkapi.AdminNavItemOpt{
		{
			Category:  sdkapi.NavCategorySystem,
			Label:     g.CoreAPI.Translate("label", "Plugins"),
			RouteName: "admin:plugins:index",
			Keywords: []string{
				g.CoreAPI.Translate("label", "plugin"),
				g.CoreAPI.Translate("label", "Plugins"),
				g.CoreAPI.Translate("label", "extension"),
				g.CoreAPI.Translate("label", "extensions"),
			},
		},
	}
}
