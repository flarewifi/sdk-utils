package handlers

import (
	"fmt"
	"log"
	"net/http"
	sdkapi "sdk/api"

	"com.flarego.default-theme/resources/views/admin"
	sdkutils "github.com/flarehotspot/sdk-utils"
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

		view := admin.NotificationsList(api, notifs)
		view.Render(r.Context(), w)
	}
}

func UpdateNotificationCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifsAPI := api.Notification()
		vars := api.Http().MuxVars(r)
		id := vars["id"]
		idInt := sdkutils.StrToInt64(id)

		var notifs []sdkapi.Notification
		if idInt == 0 {
			api.Logger().Error("No valid ID.")
		}

		err := notifsAPI.UpdateNotificationStatus(r.Context(), idInt, sdkapi.NotificationStatusRead)
		if err != nil {
			api.Logger().Error(fmt.Sprintf("update notifications error: %v", err))
		}

		notifs, err = notifsAPI.GetUnreadNotifications(r.Context())
		if err != nil {
			log.Printf("get notifications error: %v", err)
			notifs = []sdkapi.Notification{}
		}

		view := admin.NotificationForm(api, notifs)
		view.Render(r.Context(), w)
	}
}

func NotificationsBellCountCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifsAPI := api.Notification()
		vars := api.Http().MuxVars(r)
		id := vars["id"]
		idInt := sdkutils.StrToInt64(id)

		if idInt == 0 {
			api.Logger().Error("No valid ID.")
		}

		notifs, err := notifsAPI.GetUnreadNotifications(r.Context())
		if err != nil {
			api.Logger().Error(fmt.Sprintf("get notifications error: %v", err))
			notifs = []sdkapi.Notification{}
		}

		view := admin.NotificationsBellCount(notifs)
		view.Render(r.Context(), w)
	}
}

func ShowNotificationContentCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifsAPI := api.Notification()
		vars := api.Http().MuxVars(r)
		id := vars["id"]
		idInt := sdkutils.StrToInt64(id)

		if idInt == 0 {
			api.Logger().Error("No valid ID.")
		}

		notif, err := notifsAPI.GetNotificationByID(r.Context(), idInt)
		if err != nil {
			api.Logger().Error(fmt.Sprintf("get notifications error: %v", err))
		}

		view := admin.ShowNotificationContent(api, notif)
		view.Render(r.Context(), w)
	}
}
