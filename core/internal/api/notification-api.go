package api

import (
	"errors"
	sdkapi "sdk/api"
)

func NewNotificationAPI(api *PluginApi) *NotificationAPI {
	return &NotificationAPI{
		api: api,
	}
}

type NotificationAPI struct {
	api *PluginApi
}

func (n *NotificationAPI) AddNotification(notif *sdkapi.Notification) error {
	return errors.New("not yet implemented")
}

func (n *NotificationAPI) GetUnreadNotifications() (sdkapi.Notifications, error) {
	return nil, errors.New("not yet implemented")
}

func (n *NotificationAPI) UpdateNotificationStatus(id string, status sdkapi.NotificationStatus) error {
	return errors.New("not yet implemented")
}
