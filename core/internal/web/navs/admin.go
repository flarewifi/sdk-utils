package navs

import (
	"core/internal/plugins"
	"net/http"
	sdkhttp "sdk/api/http"
)

func SetAdminNavs(g *plugins.CoreGlobals) {
	coreNavs := g.CoreAPI.HttpAPI.Navs()

	coreNavs.AdminNavsFactory(func(r *http.Request) []sdkhttp.AdminNavItemOpt {
		return []sdkhttp.AdminNavItemOpt{
			{
				Category:  sdkhttp.NavCategorySystem,
				Label:     g.CoreAPI.Translate("label", "themes"),
				RouteName: "admin:themes:index",
			},
		}
	})
}
