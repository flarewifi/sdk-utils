package app

import (
	"net/http"
	sdkapi "sdk/api"
)

func SetupNavs(api sdkapi.IPluginApi) {
	api.Http().Navs().AdminNavsFactory(func(r *http.Request) []sdkapi.AdminNavItemOpt {
		return []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategoryThemes,
				Label:     api.Translate("label", "Admin Dashboard"),
				RouteName: "admin:themes:admin",
				Keywords: []string{
					api.Translate("label", "Admin Theme"),
					api.Translate("label", "Dashboard Theme"),
					api.Translate("label", "Admin Style"),
					api.Translate("label", "appearance"),
					api.Translate("label", "config"),
					api.Translate("label", "settings"),
					api.Translate("label", "layout"),
				},
				Order: 4100,
				Icon:  "<i class='bi bi-layout-text-window-reverse'></i>",
			},
			{
				Category:  sdkapi.NavCategoryThemes,
				Label:     api.Translate("label", "Captive Portal"),
				RouteName: "admin:themes:portal",
				Keywords: []string{
					api.Translate("label", "Portal Theme"),
					api.Translate("label", "Login Theme"),
					api.Translate("label", "Captive Portal Style"),
					api.Translate("label", "wifi"),
					api.Translate("label", "splash"),
					api.Translate("label", "access"),
					api.Translate("label", "authentication"),
				},
				Order: 4200,
				Icon:  "<i class='bi bi-phone'></i>",
			},
		}
	})
}
