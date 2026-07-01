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

// StartVoucherCleanupScheduler starts a daily job that:
//  1. Deletes stale activated vouchers whose sessions have already been cleaned up.
//  2. Notifies the admin about vouchers that expired before ever being used.
//
// Runs at 2:15 AM in production.
func StartVoucherCleanupScheduler(database *db.Database, mdls *models.Models, coreAPI *api.PluginApi) {
	go func() {
		if VoucherCleanupInterval > 0 {
			for {
				time.Sleep(VoucherCleanupInterval)
				performVoucherCleanup(database, mdls, coreAPI)
			}
		} else {
			for {
				now := time.Now()
				next := time.Date(now.Year(), now.Month(), now.Day(),
					VoucherCleanupHour, VoucherCleanupMinute, 0, 0, now.Location())
				if now.After(next) {
					next = next.Add(24 * time.Hour)
				}

				waitDuration := next.Sub(now)

				time.Sleep(waitDuration)
				performVoucherCleanup(database, mdls, coreAPI)
			}
		}
	}()
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

func performVoucherCleanup(database *db.Database, mdls *models.Models, coreAPI *api.PluginApi) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Delete stale activated vouchers: activated_at IS NOT NULL but session_id IS NULL
	// because the FK ON DELETE SET NULL fired when the nightly session cleanup removed
	// the parent session. The voucher served its purpose and has no further use.
	staleCount, err := database.Queries.CountStaleActivatedVouchers(ctx)
	if err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("voucher cleanup: count stale activated failed: %v", err))
		return
	}
	if staleCount > 0 {
		if err := database.Queries.DeleteStaleActivatedVouchers(ctx); err != nil {
			coreAPI.Logger().Error(fmt.Sprintf("voucher cleanup: delete stale activated failed: %v", err))
			return
		}
	}

	// Notify admin about vouchers that expired before ever being activated.
	cutoff := sql.NullTime{Time: time.Now().UTC(), Valid: true}
	expiredCount, err := database.Queries.CountExpiredUnusedVouchers(ctx, cutoff)
	if err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("voucher cleanup: count expired unused failed: %v", err))
		return
	}
	if expiredCount > 0 {
		notifyExpiredUnusedVouchers(ctx, database, coreAPI, expiredCount)
	}
}

// notifyExpiredUnusedVouchers warns the admin about vouchers that expired before use.
// Like the session cleanup, the rows are kept (not deleted), so the warning is throttled
// to avoid re-notifying on every nightly run while the condition persists.
func notifyExpiredUnusedVouchers(ctx context.Context, database *db.Database, coreAPI *api.PluginApi, count int64) {
	subject := coreAPI.Translate("warning", "Expired vouchers detected")

	throttleCutoff := time.Now().UTC().Add(-notificationThrottleWindow)
	recent, err := database.Queries.CountRecentNotificationsBySubject(ctx, queries.CountRecentNotificationsBySubjectParams{
		Subject:    subject,
		CutoffDate: throttleCutoff,
	})
	if err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("voucher cleanup: throttle check failed: %v", err))
		return
	}
	if recent > 0 {
		return
	}

	content := coreAPI.Translate("warning",
		"There are <% .count %> vouchers that expired without being used.",
		"count", count)
	if err := coreAPI.Notification().AddNotification(ctx, sdkapi.AddNotificationParams{
		Subject: subject,
		Content: content,
		Type:    sdkapi.NotificationTypeWarn,
	}); err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("voucher cleanup: add notification failed: %v", err))
	}
}
