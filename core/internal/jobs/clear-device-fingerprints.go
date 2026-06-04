package jobs

import (
	"context"
	"time"

	"core/db"
	"core/db/models"
)

func StartFingerprintCleanupScheduler(database *db.Database, mdls *models.Models) {
	go func() {
		if FingerprintCleanupInterval > 0 {
			for {
				time.Sleep(FingerprintCleanupInterval)
				performFingerprintCleanup(database, mdls)
			}
		}

		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(),
				FingerprintCleanupHour, FingerprintCleanupMinute, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}

			waitDuration := next.Sub(now)

			time.Sleep(waitDuration)
			performFingerprintCleanup(database, mdls)
		}
	}()
}

func performFingerprintCleanup(database *db.Database, mdls *models.Models) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	devicesWithExcess, err := database.Queries.GetDevicesWithExcessFingerprints(ctx)
	if err != nil {
		return
	}

	if len(devicesWithExcess) == 0 {
		return
	}

	for _, device := range devicesWithExcess {
		err := database.Queries.DeleteExcessFingerprintsForDevice(ctx, device.DeviceID)
		if err != nil {
			continue
		}
	}
}

func RunFingerprintCleanupNow(database *db.Database, mdls *models.Models) {
	performFingerprintCleanup(database, mdls)
}
