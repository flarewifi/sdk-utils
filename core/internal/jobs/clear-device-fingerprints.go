package jobs

import (
	"context"
	"fmt"
	"time"

	"core/db"
	"core/db/models"

	sdkapi "sdk/api"
)

// StartFingerprintCleanupScheduler wires up the fingerprint cleanup job. In
// production it runs daily at 3:00 AM (FingerprintCleanupInterval == 0); in
// dev it runs on a fast fixed interval instead.
func StartFingerprintCleanupScheduler(scheduler sdkapi.ISchedulerApi, database *db.Database, mdls *models.Models) error {
	fn := func(ctx context.Context) {
		performFingerprintCleanup(database, mdls)
	}

	if FingerprintCleanupInterval > 0 {
		return scheduler.Every("fingerprint-cleanup", FingerprintCleanupInterval, fn)
	}

	cron := fmt.Sprintf("%d %d * * *", FingerprintCleanupMinute, FingerprintCleanupHour)
	return scheduler.Cron("fingerprint-cleanup", cron, fn)
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
