package jobs

import (
	"context"
	"fmt"
	"time"

	"core/db"
	"core/db/models"
	"core/db/queries"
	"core/internal/api"

	sdkapi "sdk/api"
)

// StartSessionCleanupScheduler starts the nightly session cleanup job (23:30 in production).
// It deletes consumed/expired sessions and notifies the admin about sessions
// that were created over 90 days ago but never started (unredeemed vouchers).
func StartSessionCleanupScheduler(database *db.Database, mdls *models.Models, coreAPI *api.PluginApi) error {
	fn := func(ctx context.Context) {
		performSessionCleanup(database, mdls, coreAPI)
	}

	if SessionCleanupInterval > 0 {
		return coreAPI.Scheduler().Every("session-cleanup", SessionCleanupInterval, fn)
	}

	cron := fmt.Sprintf("%d %d * * *", SessionCleanupMinute, SessionCleanupHour)
	return coreAPI.Scheduler().Cron("session-cleanup", cron, fn)
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

func performSessionCleanup(database *db.Database, mdls *models.Models, coreAPI *api.PluginApi) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Delete sessions whose time/data has been fully consumed or whose expiry date has passed.
	consumedCount, err := database.Queries.CountConsumedOrExpiredSessions(ctx)
	if err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("session cleanup: count consumed/expired failed: %v", err))
		return
	}
	if consumedCount > 0 {
		if err := database.Queries.DeleteConsumedOrExpiredSessions(ctx); err != nil {
			coreAPI.Logger().Error(fmt.Sprintf("session cleanup: delete consumed/expired failed: %v", err))
			return
		}
	}

	// Notify admin about sessions created more than 90 days ago that were never started
	// (i.e. vouchers sold/created but never redeemed). These are flagged rather than
	// deleted so the admin can review and decide.
	cutoff := time.Now().UTC().AddDate(0, 0, -unusedResourceMinAgeDays)
	unstartedCount, err := database.Queries.CountUnstartedSessions(ctx, cutoff)
	if err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("session cleanup: count unstarted failed: %v", err))
		return
	}
	if unstartedCount > 0 {
		notifyUnstartedSessions(ctx, database, coreAPI, unstartedCount)
	}
}

// notifyUnstartedSessions warns the admin about long-unredeemed sessions, but throttles
// the warning: since the offending rows are kept (not deleted), the condition persists
// and would otherwise re-notify on every nightly run. We suppress a repeat if an
// identical warning already exists within notificationThrottleWindow.
func notifyUnstartedSessions(ctx context.Context, database *db.Database, coreAPI *api.PluginApi, count int64) {
	subject := coreAPI.Translate("warning", "Unstarted sessions detected")

	throttleCutoff := time.Now().UTC().Add(-unusedNotifyThrottle)
	recent, err := database.Queries.CountRecentNotificationsBySubject(ctx, queries.CountRecentNotificationsBySubjectParams{
		Subject:    subject,
		CutoffDate: throttleCutoff,
	})
	if err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("session cleanup: throttle check failed: %v", err))
		return
	}
	if recent > 0 {
		return
	}

	content := coreAPI.Translate("warning",
		"There are <% .count %> sessions created over 90 days ago that have never been started.",
		"count", count)
	if err := coreAPI.Notification().AddNotification(ctx, sdkapi.AddNotificationParams{
		Subject: subject,
		Content: content,
		Type:    sdkapi.NotificationTypeWarn,
	}); err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("session cleanup: add notification failed: %v", err))
	}
}
