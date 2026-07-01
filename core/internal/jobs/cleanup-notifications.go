package jobs

import (
	"context"
	"fmt"
	"time"

	"core/db"
	"core/db/models"
	"core/internal/api"
)

// notificationThrottleWindow is how long an identical cleanup warning (same subject)
// suppresses a repeat, so a persistent condition (e.g. unstarted sessions, expired
// vouchers) doesn't create a fresh notification on every nightly run. Shared by the
// session and voucher cleanup jobs.
const notificationThrottleWindow = 7 * 24 * time.Hour

// StartNotificationCleanupScheduler starts a daily job that caps the notifications
// table to its newest rows so it can't grow without bound. Runs at 2:00 AM in production.
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

	// Keep only the newest notifications; the retention cap lives in the SQL. A run
	// with fewer rows than the cap matches nothing and is a cheap no-op.
	if err := database.Queries.DeleteNotificationsExceedingLimit(ctx); err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("notification cleanup failed: %v", err))
	}
}
