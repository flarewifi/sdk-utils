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

		var sessionType sdkapi.SessionType
		runningSession, ok := api.SessionsMgr().RunningSession(clnt)
		if ok {
			sessionType = runningSession.Type()
		}
		summaryView := portal.SessionSummary(api, portal.SessionSummaryData{
			SessionSummary:   summary,
			IsSessionRunning: ok,
			SessionType:      sessionType,
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

func PortalStatusNavHandler(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := api.Http().Response()
		clnt, err := api.Http().GetClientDevice(r)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}
		_, isConnected := api.SessionsMgr().RunningSession(clnt)
		data := portal.PortalStatusData{
			IsSessionRunning: isConnected,
			DeviceMac:        clnt.MacAddr(),
			DeviceIP:         clnt.IpAddr(),
		}

		statusView := portal.PortalStatusNavView(api, data)
		if err := statusView.Render(r.Context(), w); err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}
	}
}
