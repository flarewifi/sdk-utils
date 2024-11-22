package themes

import (
	"net/http"
	sdkhttp "sdk/api/http"
	sdkplugin "sdk/api/plugin"

	"com.flarego.default-theme/resources/views/auth"
	"com.flarego.default-theme/resources/views/portal"
	"github.com/a-h/templ"
)

func SetPortalTheme(api sdkplugin.IPluginApi) {

	api.Themes().NewPortalTheme(sdkhttp.PortalThemeOpts{
		JsFile:  "theme.js",
		CssFile: "theme.css",
		LayoutFactory: func(w http.ResponseWriter, r *http.Request, data sdkhttp.PortalLayoutData) templ.Component {
			layout := portal.PortalLayout(data)
			return layout
		},
		LoginPageFactory: func(w http.ResponseWriter, r *http.Request, data sdkhttp.LoginPageData) sdkhttp.ViewPage {
			csrfHtml := api.Http().Helpers().CsrfHtmlTag(r)
			page := auth.LoginPage(csrfHtml, data)
			return sdkhttp.ViewPage{PageContent: page}
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request, data sdkhttp.PortalIndexData) sdkhttp.ViewPage {
			page := portal.PortalIndexPage()
			return sdkhttp.ViewPage{PageContent: page}
		},
	})

	// api.Themes().NewPortalTheme(sdkthemes.PortalTheme{
	// 	LayoutComponent: sdkthemes.ThemeComponent{
	// 		Component: "portal/ThemeLayout.vue",
	// 	},
	// 	IndexComponent: sdkthemes.ThemeComponent{
	// 		Component: "portal/ThemeIndex.vue",
	// 	},
	// 	ThemeAssets: &sdkthemes.ThemeAssets{
	// 		Styles: []string{
	// 			"vendor/bootstrap-4.6.1/bootstrap.min.css",
	//                "portal/style.css",
	// 		},
	// 	},
	// })
}
