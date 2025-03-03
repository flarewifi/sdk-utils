package controllers

import (
	"net/http"

	"core/internal/api"
	sse "core/internal/utils/sse"
)

func PortalIndexPage(g *api.CoreGlobals) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, t, err := g.PluginMgr.GetPortalTheme()
		if err != nil {
			g.CoreAPI.HttpAPI.Response().Error(w, r, err, 500)
			return
		}

		page := t.PortalTheme.IndexPageFactory(w, r)
		g.CoreAPI.HttpAPI.Response().PortalView(w, r, page)
	})
}

func PortalSseHandler(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := sse.NewSocket(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		clnt, err := g.CoreAPI.HttpAPI.GetClientDevice(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sse.AddSocket(clnt.MacAddr(), s)
		s.Listen()
	}
}
