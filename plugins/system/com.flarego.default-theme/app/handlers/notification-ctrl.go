package handlers

import (
	"fmt"
	"log"
	"net/http"
	sdkapi "sdk/api"

	"com.flarego.default-theme/resources/views/admin"
)

func TestSendNotifiCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		err := api.Notification().AddNotification(ctx, "This is a test notification", "info", sdkapi.NotificationTypeInfo)
		if err != nil {
			log.Printf("add test notification error: %v", err)
		}

		fmt.Fprintf(w, "<button hx-post='%s' hx-swap='outerHTML'>Test notif</button>", api.Http().Helpers().UrlForRoute("admin.notifications.test"))
	}
}

func NotificationsListCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifsAPI := api.Notification()
		notifs, err := notifsAPI.GetUnreadNotifications(r.Context())
		if err != nil {
			log.Printf("get notifications error: %v", err)
			notifs = []sdkapi.Notification{}
		}
		view := admin.NotificationsList(notifs)
		view.Render(r.Context(), w)
	}
}
