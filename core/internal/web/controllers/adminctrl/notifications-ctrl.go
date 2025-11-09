package adminctrl

import (
	"context"
	"core/internal/api"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	sdkapi "sdk/api"
)

func TestNotificationCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifsAPI := g.CoreAPI.NotificationAPI
		ctx := context.Background()

		err := notifsAPI.AddNotification(ctx, "This is a test notification", "This is only a test notification content", sdkapi.NotificationTypeInfo)
		if err != nil {
			log.Printf("add notification error: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}
}

func GetUnreadNotificationsCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifsAPI := g.CoreAPI.NotificationAPI
		ctx := context.Background()

		notifications, err := notifsAPI.GetUnreadNotifications(ctx)
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

		var input struct {
			ID     int64 `json:"id"`
			Status int64 `json:"status"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			res.FlashMsg(w, r, "Invalid request body", sdkapi.FlashMsgError)
			return
		}

		id := int64(input.ID)
		status := sdkapi.NotificationStatus(input.Status)

		err := notifsAPI.UpdateNotificationStatus(r.Context(), id, status)
		if err != nil {
			g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("update notifications error: %v", err))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
