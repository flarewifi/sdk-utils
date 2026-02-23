package jobs

import (
	"context"
	"log"
	"time"

	"core/db"
	"core/db/models"
	"core/utils/config"
)

const (
	// Default retention period in days (used if config is not set)
	defaultLogRetentionDays = 3
)

// StartLogCleanupScheduler starts a background goroutine that cleans up
// old logs based on configured retention period at 4AM local router time daily
func StartLogCleanupScheduler(database *db.Database, mdls *models.Models) {
	go func() {
		log.Println("[LogCleanup] Scheduler started - will run daily at 4AM")

		for {
			// Calculate duration until next 4AM
			now := time.Now()
			next4AM := time.Date(now.Year(), now.Month(), now.Day(), 4, 0, 0, 0, now.Location())
			if now.After(next4AM) {
				next4AM = next4AM.Add(24 * time.Hour)
			}

			waitDuration := next4AM.Sub(now)
			log.Printf("[LogCleanup] Next cleanup scheduled in %v (at %s)",
				waitDuration.Round(time.Second), next4AM.Format("2006-01-02 15:04:05"))

			time.Sleep(waitDuration)

			// Read retention days from application config
			retentionDays := defaultLogRetentionDays
			appCfg, err := config.ReadApplicationConfig()
			if err == nil && appCfg.LogsRetentionDays > 0 {
				retentionDays = appCfg.LogsRetentionDays
			}

			performLogCleanup(database, mdls, retentionDays)
		}
	}()
}

// performLogCleanup executes the cleanup of old logs
func performLogCleanup(database *db.Database, mdls *models.Models, retentionDays int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Printf("[LogCleanup] Starting cleanup of logs older than %d days", retentionDays)
	startTime := time.Now()

	// Get count before cleanup (for logging)
	countBefore, err := mdls.Log().CountOlderThan(ctx, retentionDays)
	if err != nil {
		log.Printf("[LogCleanup] ERROR: Failed to count old logs: %v", err)
		return
	}

	if countBefore == 0 {
		log.Printf("[LogCleanup] No logs older than %d days to clean up", retentionDays)
		return
	}

	log.Printf("[LogCleanup] Found %d log(s) older than %d days", countBefore, retentionDays)

	// Perform cleanup
	err = mdls.Log().DeleteOlderThan(ctx, retentionDays)
	if err != nil {
		log.Printf("[LogCleanup] ERROR: Failed to delete old logs: %v", err)
		return
	}

	duration := time.Since(startTime)
	log.Printf("[LogCleanup] ✓ Successfully deleted %d old log(s) in %v",
		countBefore, duration.Round(time.Millisecond))

	// Get total remaining logs (for statistics)
	totalRemaining, err := mdls.Log().CountAll(ctx)
	if err == nil {
		log.Printf("[LogCleanup] Total remaining logs: %d", totalRemaining)
	}
}

// RunLogCleanupNow executes cleanup immediately (useful for manual triggers or testing)
func RunLogCleanupNow(database *db.Database, mdls *models.Models, retentionDays int) {
	log.Printf("[LogCleanup] Manual cleanup triggered for logs older than %d days", retentionDays)
	performLogCleanup(database, mdls, retentionDays)
}
