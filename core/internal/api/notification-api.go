package api

import (
	"context"
	"encoding/json"
	"log"
	sdkapi "sdk/api"

	"core/db/models"
)

type NotificationRoutes struct {
	GetUnreadRoute string
	UpdateRoute    string
}

func NewNotificationAPI(api *PluginApi, mdl *models.Models) *NotificationAPI {
	return &NotificationAPI{
		api:    api,
		models: mdl,
	}
}

type NotificationAPI struct {
	api    *PluginApi
	models *models.Models
}

func (n *NotificationAPI) AddNotification(ctx context.Context, params sdkapi.AddNotificationParams) error {
	notif := &sdkapi.Notification{
		Subject: params.Subject,
		Content: params.Content,
		Type:    params.Type,
		Status:  sdkapi.NotificationStatusUnread,
	}

	_, err := n.models.Notification().Create(ctx, notif)
	if err != nil {
		return err
	}

	n.sendEvent(n.api, notif)

	return nil
}

func (n *NotificationAPI) GetUnreadNotifications(ctx context.Context) ([]sdkapi.Notification, error) {
	return n.models.Notification().GetUnreadNotifications(ctx)
}

func (n *NotificationAPI) UpdateNotificationStatus(ctx context.Context, id int64, status sdkapi.NotificationStatus) error {
	return n.models.Notification().UpdateNotificationStatus(ctx, id, status)
}

func (n *NotificationAPI) MarkAllAsRead(ctx context.Context) error {
	return n.models.Notification().MarkAllAsRead(ctx)
}

func (n *NotificationAPI) sendEvent(api *PluginApi, notif *sdkapi.Notification) {
	accts, err := api.AcctAPI.GetAll()
	if err != nil {
		log.Println("No accounts found:", err)
		return
	}

	data, err := json.Marshal(notif)
	if err != nil {
		log.Println("Notification json error:", err)
		return
	}

	// Send to all admin accounts
	for _, acct := range accts {
		acct.Emit(sdkapi.FlareNotificationEvent, data)
	}
}

func (n *NotificationAPI) GetNotificationByID(ctx context.Context, id int64) (sdkapi.Notification, error) {
	return n.models.Notification().GetNotificationByID(ctx, id)
}
