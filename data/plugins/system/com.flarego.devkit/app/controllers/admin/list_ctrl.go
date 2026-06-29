package admin

import (
	"net/http"

	sdkapi "sdk/api"

	"com.flarego.devkit/app/utils"
	views "com.flarego.devkit/resources/views/admin"
)

// ListCtrl renders the developer panel: the upload form plus the list of plugins
// currently installed under data/plugins/local/, flagging which are loaded right now.
func ListCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := api.Http().Response()

		plugins, err := utils.ListLocalPlugins()
		if err != nil {
			api.Logger().Error("developer: list local plugins: " + err.Error())
			plugins = []utils.LocalPlugin{}
		}

		loaded := make(map[string]bool, len(plugins))
		for _, p := range plugins {
			if _, ok := api.PluginsMgr().FindByPkg(p.Package); ok {
				loaded[p.Package] = true
			}
		}

		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: views.DeveloperListView(api, plugins, loaded),
		})
	}
}
