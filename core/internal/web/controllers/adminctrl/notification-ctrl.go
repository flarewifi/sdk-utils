package adminctrl

import (
	"core/internal/api"
	"fmt"
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

		// Fetch the notification for rendering BEFORE deleting it — opening a
		// notification marks it read, and read notifications are removed from the DB.
		notif, err := notifsAPI.GetNotificationByID(r.Context(), idInt)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := notifsAPI.DeleteNotification(r.Context(), idInt); err != nil {
			g.CoreAPI.Logger().Error(fmt.Sprintf("failed to delete read notification: %v", err))
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

		if err := notifsAPI.DeleteNotification(r.Context(), idInt); err != nil {
			g.CoreAPI.Logger().Error(fmt.Sprintf("failed to delete read notification: %v", err))
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

		if err := notifsAPI.DeleteAllNotifications(r.Context()); err != nil {
			g.CoreAPI.Logger().Error(fmt.Sprintf("failed to delete all notifications: %v", err))
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
