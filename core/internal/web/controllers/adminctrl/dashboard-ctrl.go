package adminctrl

import (
	"errors"
	"net/http"

	"core/internal/api"
)

func AdminDashboardCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, t, err := g.PluginMgr.GetAdminTheme()
		if err != nil {
			errMsg := g.CoreAPI.Translate("error", "get_admin_theme_error")
			g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(errMsg), http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())

			return
		}
		page := t.AdminTheme.IndexPageFactory(w, r)
		g.CoreAPI.HttpAPI.Response().AdminView(w, r, page)
	}
}
