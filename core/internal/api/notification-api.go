package api

import (
	"context"
	"encoding/json"
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

func (n *NotificationAPI) DeleteNotification(ctx context.Context, id int64) error {
	return n.models.Notification().DeleteNotification(ctx, id)
}

func (n *NotificationAPI) DeleteAllNotifications(ctx context.Context) error {
	return n.models.Notification().DeleteAllNotifications(ctx)
}

func (n *NotificationAPI) sendEvent(api *PluginApi, notif *sdkapi.Notification) {
	accts, err := api.AcctAPI.GetAll()
	if err != nil {
		return
	}

	data, err := json.Marshal(notif)
	if err != nil {
		return
	}

	for _, acct := range accts {
		acct.Emit(sdkapi.FlareNotificationEvent, data)
	}
}

func (n *NotificationAPI) GetNotificationByID(ctx context.Context, id int64) (sdkapi.Notification, error) {
	return n.models.Notification().GetNotificationByID(ctx, id)
}
