//go:build !mono

package main

import (
	"net/http"

	sdkapi "sdk/api"

	"com.flarego.devkit-sample/resources/views/admin"
)

func main() {}

// Init wires up a single admin page + nav entry so you can see your plugin's UI
// rendered inside the Devkit Theme. Copy this plugin as a starting point.
func Init(api sdkapi.IPluginApi) error {
	api.Http().Navs().AdminNavsFactory(func(r *http.Request) []sdkapi.AdminNavItemOpt {
		return []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     api.Translate("label", "Sample"),
				RouteName: "admin:sample:index",
				Order:     5000,
				Icon:      "<i class='bi bi-box'></i>",
			},
		}
	})

	api.Http().Router().AdminRouter(nil).Get("/sample", func(w http.ResponseWriter, r *http.Request) {
		api.Http().Response().AdminView(w, r, sdkapi.ViewPage{
			PageContent: admin.SampleAdminPage(api),
		})
	}).Name("admin:sample:index")

	return nil
}
