package navs

import (
	"core/internal/plugins"
	"net/http"
	sdkapi "sdk/api"
)

func SetAdminNavs(g *plugins.CoreGlobals) {
	coreNavs := g.CoreAPI.HttpAPI.Navs()

	coreNavs.AdminNavsFactory(func(r *http.Request) []sdkapi.AdminNavItemOpt {
		return []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "themes"),
				RouteName: "admin:themes:index",
			},
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "logs"),
				RouteName: "admin:logs:index",
			},
		}
	})
}
