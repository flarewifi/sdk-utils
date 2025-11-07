package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	sdkapi "sdk/api"

	"core/db/models"
)

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

func (n *NotificationAPI) AddNotification(ctx context.Context, notif *sdkapi.Notification) error {
	tx, err := n.api.SqlDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("unable to begin transaction: %w", err)
	}

	if _, err := n.models.Notification().Create(tx, ctx, notif); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("unable to commit: %w", err)
	}

	admin, err := n.api.AcctAPI.Find("admin")
	if err != nil {
		log.Println("No admin found:", err)
	}
	data, err := json.Marshal(map[string]string{
		"success": notif.Subject,
	})
	if err != nil {
		log.Println("Install Progress json error:", err)
	}

	admin.Emit(notif.EventName, data)

	return nil
}

func (n *NotificationAPI) GetUnreadNotifications(ctx context.Context) (sdkapi.Notifications, error) {
	tx, err := n.api.SqlDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	notifications, err := n.models.Notification().GetUnreadNotifications(tx, ctx)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		n.api.Logger().Error(err.Error())
	}

	return notifications, nil
}

func (n *NotificationAPI) UpdateNotificationStatus(ctx context.Context, id int64, status sdkapi.NotificationStatus) error {
	tx, err := n.api.SqlDB().Begin()
	if err != nil {
		return err
	}

	if err := n.models.Notification().
		UpdateNotificationStatus(tx, ctx, id, status); err != nil {
		return err
	}

	return tx.Commit()
}

func (n *NotificationAPI) GetUnreadNotificationsRoute() sdkapi.NotificationRoutes {
	return sdkapi.NotificationRoutes{
		GetUnreadRoute: n.api.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin.notification.unread"),
		UpdateRoute:    n.api.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin.notification.update"),
	}
}
