package themes

import (
	// "net/http"

	"net/http"
	sdkapi "sdk/api"

	"com.flarego.default-theme/resources/views/admin"
)

func SetAdminTheme(api sdkapi.IPluginApi) {
	api.Themes().NewAdminTheme(sdkapi.AdminThemeOpts{
		JsFile:  "theme.js",
		CssFile: "theme.css",
		CssLib:  sdkapi.CssLibBootstrap5,
		LayoutBuilder: func(w http.ResponseWriter, r *http.Request, b sdkapi.IViewBuilder) {
			head := admin.AdminHead()
			navs := api.Http().Navs().GetAdminNavs(r)
			layout := admin.AdminLayout(admin.AdminLayoutData{
				Navs:        navs,
				PageContent: b.Content(),
			})
			b.Render(head, layout)
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
			page := admin.AdminIndexPage()
			return sdkapi.ViewPage{PageContent: page}
		},
	})

	api.Http().Navs().AdminNavsFactory(func(r *http.Request) []sdkapi.AdminNavItemOpt {
		return []sdkapi.AdminNavItemOpt{
			// {
			// 	Label:       "Test",
			// 	Category:    sdkapi.NavCategorySystem,
			// 	RouteName:   "test",
			// 	RouteParams: map[string]string{"name": "test"},
			// },
		}
	})
}
