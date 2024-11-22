package controllers

import (
	"net/http"
	sdkhttp "sdk/api/http"

	"core/internal/plugins"
	sse "core/internal/utils/sse"
)

func PortalIndexPage(g *plugins.CoreGlobals) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, t, err := g.PluginMgr.GetPortalTheme()
		if err != nil {
			g.CoreAPI.HttpAPI.HttpResponse().Error(w, r, err, 500)
			return
		}

		navs := g.CoreAPI.HttpAPI.Navs().GetPortalItems(r)
		data := sdkhttp.PortalIndexData{Navs: navs}
		page := t.PortalTheme.IndexPageFactory(w, r, data)
		g.CoreAPI.HttpAPI.HttpResponse().PortalView(w, r, page)
	})
}

func PortalSseHandler(g *plugins.CoreGlobals) http.HandlerFunc {
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
