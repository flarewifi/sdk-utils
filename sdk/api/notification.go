package sdkapi

type NotificationStatus int

const (
	NotificationStatusUnread NotificationStatus = iota
	NotificationStatusRead
)

type Notification struct {
	ID      string             `json:"id"`
	Subject string             `json:"subject"`
	Content string             `json:"content"`
	Status  NotificationStatus `json:"status"`
}

type Notifications []Notification

type INotificationAPI interface {
	AddNotification(notif *Notification) error
	GetUnreadNotifications() (Notifications, error)
	UpdateNotificationStatus(id string, status NotificationStatus) error
}
