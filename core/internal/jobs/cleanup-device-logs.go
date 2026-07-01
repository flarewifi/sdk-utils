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
func StartDeviceLogCleanupScheduler(database *db.Database, mdls *models.Models, coreAPI *api.PluginApi) {
	go func() {
		if LogCleanupInterval > 0 {
			for {
				time.Sleep(LogCleanupInterval)
				performDeviceLogCleanup(database, mdls, coreAPI)
			}
		} else {
			for {
				now := time.Now()
				next := time.Date(now.Year(), now.Month(), now.Day(),
					LogCleanupHour, LogCleanupMinute, 0, 0, now.Location())
				if now.After(next) {
					next = next.Add(24 * time.Hour)
				}

				waitDuration := next.Sub(now)

				time.Sleep(waitDuration)
				performDeviceLogCleanup(database, mdls, coreAPI)
			}
		}
	}()
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
