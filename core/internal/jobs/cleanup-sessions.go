package jobs

import (
	"context"
	"log"
	"time"

	"core/db"
	"core/db/models"
)

// StartSessionCleanupScheduler starts a background goroutine that deletes
// consumed and expired sessions at end of day.
//
// Sessions are considered stale when:
//   - Time/data is fully consumed (consumption >= allocation)
//   - Expiration date has passed (started_at + exp_days < now)
//
// Deleted sessions do NOT emit EventSessionDeleted — this is intentional
// to avoid triggering cloud-sync or plugin side-effects for routine cleanup.
//
// Schedule:
//   - Dev mode: every SessionCleanupInterval
//   - Production: daily at SessionCleanupHour:SessionCleanupMinute
func StartSessionCleanupScheduler(database *db.Database, mdls *models.Models) {
	go func() {
		// Dev mode: run at fixed interval
		if SessionCleanupInterval > 0 {
			log.Printf("[SessionCleanup] DEV MODE: Running every %v", SessionCleanupInterval)
			for {
				time.Sleep(SessionCleanupInterval)
				performSessionCleanup(database, mdls)
			}
			return
		}

		// Production mode: run at specific time daily
		log.Printf("[SessionCleanup] Scheduler started - will run daily at %d:%02d",
			SessionCleanupHour, SessionCleanupMinute)

		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(),
				SessionCleanupHour, SessionCleanupMinute, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}

			waitDuration := next.Sub(now)
			log.Printf("[SessionCleanup] Next cleanup scheduled in %v (at %s)",
				waitDuration.Round(time.Second), next.Format("2006-01-02 15:04:05"))

			time.Sleep(waitDuration)
			performSessionCleanup(database, mdls)
		}
	}()
}

func performSessionCleanup(database *db.Database, mdls *models.Models) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("[SessionCleanup] Starting cleanup of consumed/expired sessions")
	startTime := time.Now()

	// Count sessions to be deleted
	count, err := database.Queries.CountConsumedOrExpiredSessions(ctx)
	if err != nil {
		log.Printf("[SessionCleanup] ERROR: Failed to count consumed/expired sessions: %v", err)
		return
	}

	if count == 0 {
		log.Println("[SessionCleanup] No consumed/expired sessions to clean up")
		return
	}

	log.Printf("[SessionCleanup] Found %d consumed/expired session(s) to delete", count)

	// Bulk delete — uses direct SQL query, does NOT emit EventSessionDeleted
	err = database.Queries.DeleteConsumedOrExpiredSessions(ctx)
	if err != nil {
		log.Printf("[SessionCleanup] ERROR: Failed to delete sessions: %v", err)
		return
	}

	duration := time.Since(startTime)
	log.Printf("[SessionCleanup] Successfully deleted %d consumed/expired session(s) in %v",
		count, duration.Round(time.Millisecond))
}
