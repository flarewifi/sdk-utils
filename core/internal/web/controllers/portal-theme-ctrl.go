package controllers

import (
	"core/internal/api"
	"net/http"

	corethemeportal "core/resources/views/themes/fallback/portal"

	sdkapi "sdk/api"
)

func PortalStatusNavCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clnt, err := g.CoreAPI.Http().GetClientDevice(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, isConnected := g.CoreAPI.SessionsMgr().RunningSession(clnt)

		view := corethemeportal.PortalStatusNavView(g.CoreAPI, isConnected, clnt.IpAddr(), clnt.MacAddr())
		if err := view.Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func PortalSessionSummaryCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		clnt, err := g.CoreAPI.Http().GetClientDevice(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		summary, err := g.CoreAPI.SessionsMgr().SessionSummary(ctx, clnt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var sessionType sdkapi.SessionType
		runningSession, ok := g.CoreAPI.SessionsMgr().RunningSession(clnt)
		if ok {
			sessionType = runningSession.Type()
		}

		view := corethemeportal.SessionSummary(g.CoreAPI, corethemeportal.SessionSummaryData{
			SessionSummary:   summary,
			IsSessionRunning: ok,
			SessionType:      sessionType,
		})
		if err := view.Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func PortalNavsCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		navs := g.CoreAPI.Http().Navs().GetPortalItems(r)
		view := corethemeportal.NavItems(g.CoreAPI, navs)
		if err := view.Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
