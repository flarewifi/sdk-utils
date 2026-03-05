package themes

import (
	"fmt"
	"net/http"
	sdkapi "sdk/api"

	"com.flarego.default-theme/app/dashboard"
	"com.flarego.default-theme/app/sysinfo"
	"com.flarego.default-theme/resources/views/admin"
)

func SetAdminTheme(api sdkapi.IPluginApi) {
	api.Themes().NewAdminTheme(sdkapi.AdminThemeOpts{
		JsFile:  "theme.js",
		CssFile: "theme.css",
		CssLib:  sdkapi.CssLibBootstrap5,
		PreviewMeta: &sdkapi.ThemePreviewMeta{
			Background:     "#f9fafb",
			PrimaryColor:   "#2563eb",
			SecondaryColor: "#3b82f6",
			AccentColor:    "#60a5fa",
			ButtonColor:    "#2563eb",
			TextColor:      "#1f2937",
		},
		LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
			navs := api.Http().Navs().GetAdminNavs(r)

			var navItems []sdkapi.AdminNavItem
			for _, nav := range navs {
				navItems = append(navItems, nav.Items...)
			}

			notifs, err := api.Notification().GetUnreadNotifications(r.Context())
			if err != nil {
				notifs = []sdkapi.Notification{}
			}

			data := admin.AdminLayoutData{
				Components:    c,
				Navs:          navs,
				NavItems:      navItems,
				Notifications: notifs,
			}
			layout := admin.AdminLayout(api, data)
			if err := layout.Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>Error rendering layout: %s</p>", err.Error())
			}
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
			fVersion := "1.0.0" // Default
			if corePlugin, ok := api.PluginsMgr().FindByPkg("com.flarego.core"); ok {
				fVersion = corePlugin.Info().Version
			}

			info, err := sysinfo.GetSystemInfo(api)
			if err != nil {
				// If there's an error, provide empty/default system info
				info = &sysinfo.SystemInfo{}
			}

			sales := dashboard.GetSalesSummaryToday(api, r.Context())
			activeData := dashboard.GetActiveUsersDataToday(api, r.Context())
			internet := dashboard.GetInternetStatus(api)
			chart := dashboard.GetRevenueChartData(api, r.Context())

			data := admin.AdminData{
				SysInfo:            info,
				Sales:              sales,
				ActiveUsersData:    activeData,
				InternetStatusData: internet,
				ChartData:          chart,
				FirmwareVersion:    fVersion,
			}

			page := admin.AdminIndexPage(api, data)
			return sdkapi.ViewPage{
				Assets: sdkapi.ViewAssets{
					JsFile:  "admin/dashboard.js",
					CssFile: "admin/dashboard.css",
				},
				PageContent: page,
			}
		},
	})
}
