package controllers

import (
	"errors"
	"net/http"

	"core/internal/api"
	sse "core/internal/utils/sse"
)

func AdminIndexPage(g *api.CoreGlobals) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, t, err := g.PluginMgr.GetAdminTheme()
		if err != nil {
			errMsg := g.CoreAPI.Translate("error", "get_admin_theme_error")
			g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(errMsg), http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())

			return
		}
		page := t.AdminTheme.IndexPageFactory(w, r)
		g.CoreAPI.HttpAPI.Response().AdminView(w, r, page)
	})
}

func AdminSseHandler(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := sse.NewSocket(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		acct, err := g.CoreAPI.HttpAPI.Auth().CurrentAcct(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sse.AddSocket(acct.Username(), s)
		s.Listen()
	}
}

func ErrorPage(w http.ResponseWriter, err error) {
	// TODO: show error page
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}
