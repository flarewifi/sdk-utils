package jobs

import (
	"context"
	"log"
	"time"

	"core/db"
	"core/db/models"
)

// StartFingerprintCleanupScheduler starts a background goroutine that cleans up
// old device fingerprints (older than 6 months).
// In dev mode: runs every 5 seconds. In prod: runs daily at 3:00 AM.
func StartFingerprintCleanupScheduler(database *db.Database, mdls *models.Models) {
	go func() {
		// Dev mode: run at fixed interval
		if FingerprintCleanupInterval > 0 {
			log.Printf("[FingerprintCleanup] DEV MODE: Running every %v", FingerprintCleanupInterval)
			for {
				time.Sleep(FingerprintCleanupInterval)
				performFingerprintCleanup(database, mdls)
			}
		}

		// Production mode: run at specific time daily
		log.Printf("[FingerprintCleanup] Scheduler started - will run daily at %d:%02d AM",
			FingerprintCleanupHour, FingerprintCleanupMinute)

		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(),
				FingerprintCleanupHour, FingerprintCleanupMinute, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}

			waitDuration := next.Sub(now)
			log.Printf("[FingerprintCleanup] Next cleanup scheduled in %v (at %s)",
				waitDuration.Round(time.Second), next.Format("2006-01-02 15:04:05"))

			time.Sleep(waitDuration)
			performFingerprintCleanup(database, mdls)
		}
	}()
}

// performFingerprintCleanup removes excess fingerprints per device,
// keeping only the 10 most recent (by last_seen_at).
func performFingerprintCleanup(database *db.Database, mdls *models.Models) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("[FingerprintCleanup] Starting cleanup (keeping max %d fingerprints per device)",
		MaxFingerprintsPerDevice)
	startTime := time.Now()

	// Find devices with more fingerprints than the limit (hardcoded to 10 in SQL)
	devicesWithExcess, err := database.Queries.GetDevicesWithExcessFingerprints(ctx)
	if err != nil {
		log.Printf("[FingerprintCleanup] ERROR: Failed to find devices with excess fingerprints: %v", err)
		return
	}

	if len(devicesWithExcess) == 0 {
		log.Println("[FingerprintCleanup] No devices with excess fingerprints")
		return
	}

	log.Printf("[FingerprintCleanup] Found %d device(s) with excess fingerprints", len(devicesWithExcess))

	// Delete excess fingerprints for each device
	totalDeleted := int64(0)
	for _, device := range devicesWithExcess {
		excessCount := device.FingerprintCount - int64(MaxFingerprintsPerDevice)
		err := database.Queries.DeleteExcessFingerprintsForDevice(ctx, device.DeviceID)
		if err != nil {
			log.Printf("[FingerprintCleanup] WARN: Failed to cleanup device %d: %v", device.DeviceID, err)
			continue
		}
		totalDeleted += excessCount
	}

	duration := time.Since(startTime)
	log.Printf("[FingerprintCleanup] Deleted %d excess fingerprint(s) from %d device(s) in %v",
		totalDeleted, len(devicesWithExcess), duration.Round(time.Millisecond))

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

// trigger rebuild
