package navs

import (
	"net/http"

	sdkapi "sdk/api"
)

func SetAdminNavs(api sdkapi.IPluginApi) {
	api.Http().Navs().AdminNavsFactory(func(r *http.Request) []sdkapi.AdminNavItemOpt {
		return []sdkapi.AdminNavItemOpt{
			{
				Label:     api.Translate("label", "Developer"),
				Category:  sdkapi.NavCategorySystem,
				RouteName: "admin:developer:index",
				Keywords: []string{
					"developer", "plugin", "upload", "install", "devel", "sdk",
					api.Translate("label", "Developer"),
				},
				Icon: "<i class='bi bi-code-square'></i>",
			},
		}
	})
}
