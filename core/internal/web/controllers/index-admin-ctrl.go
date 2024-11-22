package controllers

import (
	"net/http"

	"core/internal/plugins"
	sse "core/internal/utils/sse"
)

func AdminIndexPage(g *plugins.CoreGlobals) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, t, err := g.PluginMgr.GetAdminTheme()
		if err != nil {
			g.CoreAPI.HttpAPI.HttpResponse().Error(w, r, err, 500)
			return
		}
		page := t.AdminTheme.IndexPageFactory(w, r)
		g.CoreAPI.HttpAPI.HttpResponse().AdminView(w, r, page)
	})
}

func AdminSseHandler(g *plugins.CoreGlobals) http.HandlerFunc {
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
