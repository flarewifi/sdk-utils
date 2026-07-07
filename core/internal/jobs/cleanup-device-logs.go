package jobs

import (
	"context"
	"fmt"
	"time"

	"core/db"
	"core/db/models"
	"core/internal/api"
)

// StartDeviceLogCleanupScheduler wires up the device log cleanup job.
// In production LogCleanupInterval is 1h (interval-based); in dev it runs every 500s.
func StartDeviceLogCleanupScheduler(database *db.Database, mdls *models.Models, coreAPI *api.PluginApi) error {
	fn := func(ctx context.Context) {
		performDeviceLogCleanup(database, mdls, coreAPI)
	}

	if LogCleanupInterval > 0 {
		return coreAPI.Scheduler().Every("device-log-cleanup", LogCleanupInterval, fn)
	}

	cron := fmt.Sprintf("%d %d * * *", LogCleanupMinute, LogCleanupHour)
	return coreAPI.Scheduler().Cron("device-log-cleanup", cron, fn)
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

func performDeviceLogCleanup(database *db.Database, mdls *models.Models, coreAPI *api.PluginApi) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// DeleteOldDeviceLogs removes logs older than 90 days (cutoff hardcoded in SQL).
	if err := database.Queries.DeleteOldDeviceLogs(ctx); err != nil {
		coreAPI.Logger().Error(fmt.Sprintf("device log cleanup failed: %v", err))
	}
}
