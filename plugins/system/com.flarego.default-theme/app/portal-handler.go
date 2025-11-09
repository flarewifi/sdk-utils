package app

import (
	"net/http"
	sdkapi "sdk/api"

	"com.flarego.default-theme/resources/views/portal"
)

func PortalSessionSyncHandler(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := api.Http().Response()
		ctx := r.Context()
		clnt, err := api.Http().GetClientDevice(r)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		summary, err := api.SessionsMgr().SessionSummary(ctx, clnt)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		_, ok := api.SessionsMgr().CurrSession(clnt)
		summaryView := portal.SessionSummary(api, portal.SessionSummaryData{
			SessionSummary:   summary,
			IsSessionRunning: ok,
		})

		if err := summaryView.Render(r.Context(), w); err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}
	}
}

func PortalNavItemsHandler(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		navs := api.Http().Navs().GetPortalItems(r)
		navsView := portal.NavItems(api, navs)
		navsView.Render(r.Context(), w)
	}
}
