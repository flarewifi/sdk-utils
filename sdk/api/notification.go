package sdkapi

import (
	"context"
	"time"
)

type NotificationStatus int
type NotificationType string

const (
	NotificationStatusUnread NotificationStatus = iota
	NotificationStatusRead
)

const (
	FlareNotificationEvent  string           = "flare_notification"
	NotificationTypeSuccess NotificationType = "success"
	NotificationTypeError   NotificationType = "error"
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeWarn    NotificationType = "warn"
)

type Notification struct {
	ID        int64              `json:"id"`
	Subject   string             `json:"subject"`
	Content   string             `json:"content"`
	Status    NotificationStatus `json:"status"`
	Type      NotificationType   `json:"type"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type INotificationAPI interface {
	AddNotification(ctx context.Context, subject string, content string, t NotificationType) error
	GetUnreadNotifications(ctx context.Context) ([]Notification, error)
	GetNotificationByID(ctx context.Context, id int64) (Notification, error)
	UpdateNotificationStatus(ctx context.Context, id int64, status NotificationStatus) error
}
