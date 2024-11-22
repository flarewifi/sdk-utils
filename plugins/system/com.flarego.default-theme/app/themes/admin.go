package themes

import (
	// "net/http"

	"net/http"
	sdkhttp "sdk/api/http"
	plugin "sdk/api/plugin"

	"com.flarego.default-theme/resources/views/admin"
	"github.com/a-h/templ"
)

func SetAdminTheme(api plugin.IPluginApi) {
	api.Themes().NewAdminTheme(sdkhttp.AdminThemeOpts{
		JsFile:  "theme.js",
		CssFile: "theme.css",
		CssLib:  sdkhttp.CssLibBootstrap5,
		LayoutFactory: func(w http.ResponseWriter, r *http.Request, data sdkhttp.AdminLayoutData) templ.Component {
			layout := admin.AdminLayout(api, data)
			return layout
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkhttp.ViewPage {
			page := admin.AdminIndexPage()
			return sdkhttp.ViewPage{PageContent: page}
		},
	})

	api.Http().Navs().AdminNavsFactory(func(r *http.Request) []sdkhttp.AdminNavItemOpt {
		return []sdkhttp.AdminNavItemOpt{
			{
				Label:     "Test",
				Category:  sdkhttp.NavCategorySystem,
				RouteName: "test",
			},
		}
	})

	// api.Themes().NewAdminTheme(themes.AdminTheme{
	// 	CssLib: themes.CssLibBootstrap4,
	// 	DashboardComponent: themes.ThemeComponent{
	// 		RouteName: "dashboard",
	// 		Component: "admin/Dashboard.vue",
	// 	},
	// 	LayoutComponent: themes.ThemeComponent{
	// 		Component: "admin/ThemeLayout.vue",
	// 	},
	// 	LoginComponent: themes.ThemeComponent{
	// 		RouteName: "login",
	// 		Component: "admin/ThemeLogin.vue",
	// 	},
	// 	ThemeAssets: &themes.ThemeAssets{
	// 		Scripts: []string{
	// 			"vendor/polyfills/intersection-observer.js",
	// 			"vendor/polyfills/intersection-observer-enable-polling.js",
	// 			"vendor/bootstrap-vue/bootstrap-vue-2.23.1.umd.min.js",
	// 			"vendor/bootstrap-vue/bootstrap-vue-icons-2.23.1.umd.min.js",
	// 		},
	// 		Styles: []string{
	// 			"vendor/bootstrap-4.6.1/bootstrap.min.css",
	// 		},
	// 	},
	// })
}
