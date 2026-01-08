package jobs

import (
	"context"
	"log"
	"time"

	"core/db"
	"core/db/models"
)

// StartFingerprintCleanupScheduler starts a background goroutine that cleans up
// old device fingerprints (older than 6 months) at 3AM local router time daily
func StartFingerprintCleanupScheduler(database *db.Database, mdls *models.Models) {
	go func() {
		log.Println("[FingerprintCleanup] Scheduler started - will run daily at 3AM")

		for {
			// Calculate duration until next 3AM
			now := time.Now()
			next3AM := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
			if now.After(next3AM) {
				next3AM = next3AM.Add(24 * time.Hour)
			}

			waitDuration := next3AM.Sub(now)
			log.Printf("[FingerprintCleanup] Next cleanup scheduled in %v (at %s)",
				waitDuration.Round(time.Second), next3AM.Format("2006-01-02 15:04:05"))

			time.Sleep(waitDuration)
			performFingerprintCleanup(database, mdls)
		}
	}()
}

// performFingerprintCleanup executes the cleanup of old fingerprints
func performFingerprintCleanup(database *db.Database, mdls *models.Models) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("[FingerprintCleanup] Starting cleanup of fingerprints older than 6 months")
	startTime := time.Now()

	// Get count before cleanup (for logging)
	beforeQuery := `SELECT COUNT(*) FROM device_fingerprints WHERE created_at < datetime('now', '-6 months')`
	var countBefore int64
	err := database.DB.QueryRowContext(ctx, beforeQuery).Scan(&countBefore)
	if err != nil {
		log.Printf("[FingerprintCleanup] ERROR: Failed to count old fingerprints: %v", err)
		return
	}

	if countBefore == 0 {
		log.Println("[FingerprintCleanup] No old fingerprints to clean up")
		return
	}

	log.Printf("[FingerprintCleanup] Found %d fingerprint(s) older than 6 months", countBefore)

	// Perform cleanup
	err = mdls.DeviceFingerprint().DeleteOldFingerprints(ctx)
	if err != nil {
		log.Printf("[FingerprintCleanup] ERROR: Failed to delete old fingerprints: %v", err)
		return
	}

	duration := time.Since(startTime)
	log.Printf("[FingerprintCleanup] ✓ Successfully deleted %d old fingerprint(s) in %v",
		countBefore, duration.Round(time.Millisecond))

	// Get total remaining fingerprints (for statistics)
	totalQuery := `SELECT COUNT(*) FROM device_fingerprints`
	var totalRemaining int64
	err = database.DB.QueryRowContext(ctx, totalQuery).Scan(&totalRemaining)
	if err == nil {
		log.Printf("[FingerprintCleanup] Total remaining fingerprints: %d", totalRemaining)
	}
}

// RunFingerprintCleanupNow executes cleanup immediately (useful for manual triggers or testing)
func RunFingerprintCleanupNow(database *db.Database, mdls *models.Models) {
	log.Println("[FingerprintCleanup] Manual cleanup triggered")
	performFingerprintCleanup(database, mdls)
}
