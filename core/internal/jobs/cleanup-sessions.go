package jobs

import (
	"context"
	"time"

	"core/db"
	"core/db/models"
)

func StartSessionCleanupScheduler(database *db.Database, mdls *models.Models) {
	go func() {
		if SessionCleanupInterval > 0 {
			for {
				time.Sleep(SessionCleanupInterval)
				performSessionCleanup(database, mdls)
			}
		} else {
			for {
				now := time.Now()
				next := time.Date(now.Year(), now.Month(), now.Day(),
					SessionCleanupHour, SessionCleanupMinute, 0, 0, now.Location())
				if now.After(next) {
					next = next.Add(24 * time.Hour)
				}

				waitDuration := next.Sub(now)

				time.Sleep(waitDuration)
				performSessionCleanup(database, mdls)
			}
		}
	}()
}

func performSessionCleanup(database *db.Database, mdls *models.Models) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	count, err := database.Queries.CountConsumedOrExpiredSessions(ctx)
	if err != nil {
		return
	}

	if count == 0 {
		return
	}

	err = database.Queries.DeleteConsumedOrExpiredSessions(ctx)
	if err != nil {
		return
	}
}
