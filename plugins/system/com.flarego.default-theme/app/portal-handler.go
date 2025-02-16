package app

import (
	"fmt"
	"net/http"
	sdkapi "sdk/api"

	"com.flarego.default-theme/resources/views/portal"
)

func PortalSessionSyncHandler(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := api.Http().Response()
		clnt, err := api.Http().GetClientDevice(r)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		summary, err := api.SessionsMgr().SessionSummary(r.Context(), clnt)
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
			fmt.Println("Error rendering session summary: ", err)
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

func TriggerSessionSync(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := api.Http().Response()
		clnt, err := api.Http().GetClientDevice(r)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		clnt.Emit(portal.EventSessionSync, []byte(""))
		w.WriteHeader(http.StatusOK)
	}
}
