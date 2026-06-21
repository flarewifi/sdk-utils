package adminctrl

import (
	"core/internal/api"
	"net/http"

	corethemeadmin "core/resources/views/themes/fallback/admin"

	sdkutils "github.com/flarewifi/sdk-utils"
	sdkapi "sdk/api"
)

func NotificationsListCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifsAPI := g.CoreAPI.Notification()
		notifs, err := notifsAPI.GetUnreadNotifications(r.Context())
		if err != nil {
			notifs = []sdkapi.Notification{}
		}

		view := corethemeadmin.NotificationsList(g.CoreAPI, notifs)
		view.Render(r.Context(), w)
	}
}

func NotificationsBellCountCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifsAPI := g.CoreAPI.Notification()
		notifs, err := notifsAPI.GetUnreadNotifications(r.Context())
		if err != nil {
			notifs = []sdkapi.Notification{}
		}

		view := corethemeadmin.NotificationsBellCount(notifs)
		view.Render(r.Context(), w)
	}
}

func ShowNotificationContentCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifsAPI := g.CoreAPI.Notification()
		vars := g.CoreAPI.Http().MuxVars(r)
		id := vars["id"]
		idInt := sdkutils.StrToInt64(id)

		if idInt == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := notifsAPI.UpdateNotificationStatus(r.Context(), idInt, sdkapi.NotificationStatusRead); err != nil {
		}

		notif, err := notifsAPI.GetNotificationByID(r.Context(), idInt)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("HX-Trigger", "notificationRead")
		view := corethemeadmin.ShowNotificationContent(g.CoreAPI, notif)
		view.Render(r.Context(), w)
	}
}

func UpdateNotificationCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifsAPI := g.CoreAPI.Notification()
		vars := g.CoreAPI.Http().MuxVars(r)
		id := vars["id"]
		idInt := sdkutils.StrToInt64(id)

		if idInt == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := notifsAPI.UpdateNotificationStatus(r.Context(), idInt, sdkapi.NotificationStatusRead); err != nil {
		}

		notifs, err := notifsAPI.GetUnreadNotifications(r.Context())
		if err != nil {
			notifs = []sdkapi.Notification{}
		}

		w.Header().Set("HX-Trigger", "notificationRead")
		view := corethemeadmin.NotificationsList(g.CoreAPI, notifs)
		view.Render(r.Context(), w)
	}
}

func ClearAllNotificationsCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifsAPI := g.CoreAPI.Notification()

		if err := notifsAPI.MarkAllAsRead(r.Context()); err != nil {
		}

		notifs, err := notifsAPI.GetUnreadNotifications(r.Context())
		if err != nil {
			notifs = []sdkapi.Notification{}
		}

		w.Header().Set("HX-Trigger", "notificationRead")
		view := corethemeadmin.NotificationsList(g.CoreAPI, notifs)
		view.Render(r.Context(), w)
	}
}
