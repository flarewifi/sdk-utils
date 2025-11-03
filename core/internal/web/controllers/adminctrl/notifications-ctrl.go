package adminctrl

import (
	"core/internal/api"
	"encoding/json"
	"fmt"
	"net/http"
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/gorilla/mux"
)

func GetUnreadNotificationsCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifsAPI := g.CoreAPI.NotificationAPI

		notifications, err := notifsAPI.GetUnreadNotifications(r.Context())
		if err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("get notifications error: %v", err))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"notifications": notifications,
		})
	}
}

func UpdateNotificationCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.Http().Response()
		notifsAPI := api.NotificationAPI

		vars := mux.Vars(r)
		idStr := vars["id"]
		if idStr == "" {
			res.FlashMsg(w, r, "Unable to update notification", sdkapi.FlashMsgError)
			return
		}

		params := r.URL.Query()
		vals, ok := params["status"]
		if !ok || len(vals) == 0 {
			res.FlashMsg(w, r, "Status not found", sdkapi.FlashMsgError)
			return
		}

		id := int64(sdkutils.AtoiOrDefault(idStr, 0))
		status := sdkapi.NotificationStatus(sdkutils.AtoiOrDefault(vals[0], 0))

		err := notifsAPI.UpdateNotificationStatus(r.Context(), id, status)
		if err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("update notifications error: %v", err))
			return
		}

		res.FlashMsg(w, r, "Notification has been updated.", sdkapi.FlashMsgSuccess)
	}
}
