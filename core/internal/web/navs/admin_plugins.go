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
			Label:     g.CoreAPI.Translate("label", "plugins"),
			RouteName: "admin:plugins:index",
			Keywords:  []string{"plugin", "plugins", "extension", "extensions"},
		},
	}
}
