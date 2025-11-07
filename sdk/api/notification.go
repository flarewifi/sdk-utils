package sdkapi

import (
	"context"
	"time"
)

type NotificationStatus int
type EventStatus int

const (
	NotificationStatusUnread NotificationStatus = iota
	NotificationStatusRead
)

const (
	EventStatusSuccess EventStatus = iota
	EventStatusFailed
)

type Notification struct {
	ID          int64              `json:"id"`
	Subject     string             `json:"subject"`
	Content     string             `json:"content"`
	Status      NotificationStatus `json:"status"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	EventName   string
	EventStatus EventStatus
}

type NotificationRoutes struct {
	GetUnreadRoute string
	UpdateRoute    string
}

type Notifications []Notification

type INotificationAPI interface {
	AddNotification(ctx context.Context, notif *Notification) error
	GetUnreadNotifications(ctx context.Context) (Notifications, error)
	UpdateNotificationStatus(ctx context.Context, id int64, status NotificationStatus) error
	GetUnreadNotificationsRoute() NotificationRoutes
}
