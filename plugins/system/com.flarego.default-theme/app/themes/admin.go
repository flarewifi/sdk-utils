package themes

import (
	"fmt"
	"log"
	"net/http"
	sdkapi "sdk/api"

	"com.flarego.default-theme/resources/views/admin"
)

func SetAdminTheme(api sdkapi.IPluginApi) {
	api.Themes().NewAdminTheme(sdkapi.AdminThemeOpts{
		JsFile:  "theme.js",
		CssFile: "theme.css",
		CssLib:  sdkapi.CssLibBootstrap5,
		LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
			navs := api.Http().Navs().GetAdminNavs(r)
			notifRoutes := api.Notification().GetUnreadNotificationsRoute()
			log.Println("notifRoutes: ", notifRoutes)

			var navItems []sdkapi.AdminNavItem
			for _, nav := range navs {
				navItems = append(navItems, nav.Items...)
			}

			data := admin.AdminLayoutData{
				Components:         c,
				Navs:               navs,
				NavItems:           navItems,
				NotificationRoutes: notifRoutes,
			}
			layout := admin.AdminLayout(api, data)
			if err := layout.Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>Error rendering layout: %s</p>", err.Error())
			}
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
			page := admin.AdminIndexPage()
			return sdkapi.ViewPage{PageContent: page}
		},
	})
}
