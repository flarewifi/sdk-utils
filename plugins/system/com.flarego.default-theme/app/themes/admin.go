package themes

import (
	"fmt"
	"net/http"
	sdkapi "sdk/api"

	"com.flarego.default-theme/app/sysinfo"
	"com.flarego.default-theme/resources/views/admin"
)

func SetAdminTheme(api sdkapi.IPluginApi) {
	api.Themes().NewAdminTheme(sdkapi.AdminThemeOpts{
		JsFile:  "theme.js",
		CssFile: "theme.css",
		CssLib:  sdkapi.CssLibBootstrap5,
		LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
			navs := api.Http().Navs().GetAdminNavs(r)

			var navItems []sdkapi.AdminNavItem
			for _, nav := range navs {
				navItems = append(navItems, nav.Items...)
			}

			data := admin.AdminLayoutData{
				Components: c,
				Navs:       navs,
				NavItems:   navItems,
			}
			layout := admin.AdminLayout(api, data)
			if err := layout.Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>Error rendering layout: %s</p>", err.Error())
			}
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
			// Get system information
			info, err := sysinfo.GetSystemInfo()
			if err != nil {
				// If there's an error, provide empty/default system info
				info = &sysinfo.SystemInfo{}
			}

			page := admin.AdminIndexPage(api, info)
			return sdkapi.ViewPage{PageContent: page}
		},
	})
}
