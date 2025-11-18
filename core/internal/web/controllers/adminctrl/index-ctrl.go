package adminctrl

import (
	"errors"
	"net/http"

	"core/internal/api"
)

func AdminIndexCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, t, err := g.PluginMgr.GetAdminTheme()
		if err != nil {
			errMsg := g.CoreAPI.Translate("error", "Unable to Get Admin Theme")
			g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(errMsg), http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())

			return
		}
		page := t.AdminTheme.IndexPageFactory(w, r)
		p.Http().Response().AdminView(w, r, page)
	}
}
