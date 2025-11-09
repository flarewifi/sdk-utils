package models

import (
	"context"
	"core/db"
	"core/db/queries"
	"database/sql"
	"fmt"
	sdkapi "sdk/api"
	"time"
)

type NotificationModel struct {
	db     *db.Database
	models *Models
}

func NewNotificationModel(dtb *db.Database, mdls *Models) *NotificationModel {
	return &NotificationModel{dtb, mdls}
}

func (nm *NotificationModel) Create(ctx context.Context, notif *sdkapi.Notification) (int64, error) {
	q := nm.db.Queries
	id, err := q.CreateNotification(ctx, queries.CreateNotificationParams{
		Subject: notif.Subject,
		Content: notif.Content,
		Status:  int64(notif.Status),
		Type:    string(notif.Type),
	})
	if err != nil {
		return 0, fmt.Errorf("create notification error: %w", err)
	}

	return id, nil
}

func (nm *NotificationModel) GetUnreadNotifications(ctx context.Context) ([]sdkapi.Notification, error) {
	q := nm.db.Queries

	dbNotifs, err := q.GetUnreadNotifications(ctx, int64(sdkapi.NotificationStatusUnread))
	if err != nil {
		return nil, fmt.Errorf("get unread notifications error: %w", err)
	}

	notifications := make([]sdkapi.Notification, len(dbNotifs))
	for i, n := range dbNotifs {
		notifications[i] = sdkapi.Notification{
			ID:        n.ID,
			Subject:   n.Subject,
			Content:   n.Content,
			Status:    sdkapi.NotificationStatus(n.Status),
			CreatedAt: n.CreatedAt.In(time.Local),
			UpdatedAt: n.UpdatedAt.In(time.Local),
		}
	}

	return notifications, nil
}

func (nm *NotificationModel) UpdateNotificationStatus(tx *sql.Tx, ctx context.Context, id int64, status sdkapi.NotificationStatus) error {
	qtx := nm.db.Queries.WithTx(tx)

	return qtx.UpdateNotificationStatus(ctx, queries.UpdateNotificationStatusParams{
		Status: int64(status),
		ID:     id,
	})
}
