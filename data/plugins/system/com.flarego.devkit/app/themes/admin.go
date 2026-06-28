package themes

import (
	"fmt"
	"net/http"

	sdkapi "sdk/api"

	"com.flarego.devkit/resources/views/admin"
)

// SetAdminTheme registers the Devkit admin theme: a Bootstrap 5 top-nav layout
// that hosts the developer's plugin pages and renders their admin nav items.
func SetAdminTheme(api sdkapi.IPluginApi) {
	api.Themes().NewAdminTheme(sdkapi.AdminThemeOpts{
		JsFile:       "theme.js",
		CssFile:      "theme.css",
		CssLib:       sdkapi.CssLibBootstrap5,
		PreviewImage: "images/devkit-admin.svg",
		LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
			navs := api.Http().Navs().GetAdminNavs(r)

			notifs, err := api.Notification().GetUnreadNotifications(r.Context())
			if err != nil {
				notifs = []sdkapi.Notification{}
			}

			data := admin.AdminLayoutData{
				Components:    c,
				Navs:          navs,
				Notifications: notifs,
			}
			layout := admin.AdminLayout(api, data)
			if err := layout.Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>Error rendering layout: %s</p>", err.Error())
			}
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
			return sdkapi.ViewPage{
				Assets:      sdkapi.ViewAssets{},
				PageContent: admin.AdminIndexPage(api),
			}
		},
	})
}
