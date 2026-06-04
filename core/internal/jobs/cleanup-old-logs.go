package jobs

import (
	"context"
	"time"

	"core/db"
	"core/db/models"
	"core/utils/config"
)

const (
	defaultLogRetentionDays = 3
)

func StartLogCleanupScheduler(database *db.Database, mdls *models.Models) {
	go func() {
		for {
			time.Sleep(LogCleanupInterval)

			retentionDays := defaultLogRetentionDays
			appCfg, err := config.ReadApplicationConfig()
			if err == nil && appCfg.LogsRetentionDays > 0 {
				retentionDays = appCfg.LogsRetentionDays
			}

			performLogCleanup(database, mdls, retentionDays)
		}
	}()
}

func performLogCleanup(database *db.Database, mdls *models.Models, retentionDays int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	countBefore, err := mdls.Log().CountOlderThan(ctx, retentionDays)
	if err != nil {
		return
	}

	if countBefore != 0 {
		err = mdls.Log().DeleteOlderThan(ctx, retentionDays)
		if err != nil {
			return
		}
	}
}

func RunLogCleanupNow(database *db.Database, mdls *models.Models, retentionDays int) {
	performLogCleanup(database, mdls, retentionDays)
}
