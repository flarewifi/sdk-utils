package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"core/db"
	"core/db/models"
	"core/db/queries"
	"core/internal/api"

	sdkapi "sdk/api"
)

// StartNotificationCleanupScheduler starts a daily job that ages out long-read
// notifications and caps the unread pile-up so the table can't grow without bound.
// Runs at 2:00 AM in production.
func StartNotificationCleanupScheduler(database *db.Database, mdls *models.Models, coreAPI *api.PluginApi) {
	go func() {
		if NotificationCleanupInterval > 0 {
			for {
				time.Sleep(NotificationCleanupInterval)
				performNotificationCleanup(database, mdls, coreAPI)
			}
		} else {
			for {
				now := time.Now()
				next := time.Date(now.Year(), now.Month(), now.Day(),
					NotificationCleanupHour, NotificationCleanupMinute, 0, 0, now.Location())
				if now.After(next) {
					next = next.Add(24 * time.Hour)
				}

				waitDuration := next.Sub(now)

				time.Sleep(waitDuration)
				performNotificationCleanup(database, mdls, coreAPI)
			}
		}
	}()
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

func performNotificationCleanup(database *db.Database, mdls *models.Models, coreAPI *api.PluginApi) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Standardized retention: delete notifications that have been READ for more than
	// the retention window. Unread rows (including the daily unused-resource warnings)
	// are untouched here — they are only bounded by the backstop cap below.
	readCutoff := sql.NullTime{Time: time.Now().UTC().AddDate(0, 0, -readNotificationRetentionDays), Valid: true}
	if err := database.Queries.DeleteReadNotificationsOlderThan(ctx, queries.DeleteReadNotificationsOlderThanParams{
		Status:     int64(sdkapi.NotificationStatusRead),
		CutoffDate: readCutoff,
	}); err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("notification cleanup: delete long-read failed: %v", err))
	}

	// Backstop: cap the unread pile-up so a persistent daily-warning condition can't
	// grow the table without bound. A run with fewer rows than the cap is a no-op.
	if err := database.Queries.DeleteNotificationsExceedingLimit(ctx); err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("notification cleanup: cap failed: %v", err))
	}
}
