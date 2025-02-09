package themes

import (
	// "net/http"

	"net/http"
	sdkapi "sdk/api"

	"com.flarego.default-theme/resources/views/admin"
	"com.flarego.default-theme/resources/views/flash"
)

func SetAdminTheme(api sdkapi.IPluginApi) {
	api.Themes().NewAdminTheme(sdkapi.AdminThemeOpts{
		JsFile:  "theme.js",
		CssFile: "theme.css",
		CssLib:  sdkapi.CssLibBootstrap5,
		LayoutFactory: func(w http.ResponseWriter, r *http.Request, data sdkapi.AdminLayoutData) {
			head := admin.AdminHead()
			flashMsg := data.Builder.FlashMsg()
			flashTpl := flash.FlashMsg(flashMsg)
			layout := admin.AdminLayout(admin.AdminLayoutData{
				Flash:       flashTpl,
				Navs:        data.Navs,
				PageContent: data.Builder.Content(),
			})
			data.Builder.Render(head, layout)
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
			page := admin.AdminIndexPage()
			return sdkapi.ViewPage{PageContent: page}
		},
	})

	api.Http().Navs().AdminNavsFactory(func(r *http.Request) []sdkapi.AdminNavItemOpt {
		return []sdkapi.AdminNavItemOpt{
			{
				Label:       "Test",
				Category:    sdkapi.NavCategorySystem,
				RouteName:   "test",
				RouteParams: map[string]string{"name": "test"},
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
