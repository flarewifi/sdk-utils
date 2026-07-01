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

	// Delete USED (activated) vouchers past the 30-day retention window, keyed on
	// activated_at. Decoupled from the parent session: a used voucher is kept 30 days
	// from activation whether or not its session still exists.
	usedCutoff := sql.NullTime{Time: time.Now().UTC().AddDate(0, 0, -usedResourceRetentionDays), Valid: true}
	usedCount, err := database.Queries.CountUsedVouchers(ctx, usedCutoff)
	if err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("voucher cleanup: count used failed: %v", err))
		return
	}
	if usedCount > 0 {
		if err := database.Queries.DeleteUsedVouchers(ctx, usedCutoff); err != nil {
			coreAPI.Logger().Error(fmt.Sprintf("voucher cleanup: delete used failed: %v", err))
			return
		}
	}

	// Notify admin about UNUSED (never activated) vouchers older than 90 days. These
	// are kept (never auto-deleted) so the admin can review; the warning recurs daily.
	unusedCutoff := sql.NullTime{Time: time.Now().UTC().AddDate(0, 0, -unusedResourceMinAgeDays), Valid: true}
	unusedCount, err := database.Queries.CountUnusedVouchers(ctx, unusedCutoff)
	if err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("voucher cleanup: count unused failed: %v", err))
		return
	}
	if unusedCount > 0 {
		notifyUnusedVouchers(ctx, database, coreAPI, unusedCount)
	}
}

// notifyUnusedVouchers warns the admin about vouchers created long ago that were
// never activated. The rows are kept (not deleted), so the warning is throttled to
// fire at most once per day while the condition persists.
func notifyUnusedVouchers(ctx context.Context, database *db.Database, coreAPI *api.PluginApi, count int64) {
	subject := coreAPI.Translate("warning", "Unused vouchers detected")

	throttleCutoff := time.Now().UTC().Add(-unusedNotifyThrottle)
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
		"There are <% .count %> vouchers created over 90 days ago that were never used.",
		"count", count)
	if err := coreAPI.Notification().AddNotification(ctx, sdkapi.AddNotificationParams{
		Subject: subject,
		Content: content,
		Type:    sdkapi.NotificationTypeWarn,
	}); err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("voucher cleanup: add notification failed: %v", err))
	}
}
