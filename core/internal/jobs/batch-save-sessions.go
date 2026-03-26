package jobs

import (
	"context"
	"log"
	"time"

	"core/internal/sessmgr"
)

// StartBatchSaveLoop launches a background goroutine that periodically flushes
// all running sessions to the database via SessionsMgr.FlushRunningSessions.
// Returns a cancel function to stop the loop (performs a final flush on stop).
func StartBatchSaveLoop(clientMgr *sessmgr.SessionsMgr) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(BatchSaveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				clientMgr.FlushRunningSessions()
				log.Println("[BatchSave] Batch save loop stopped")
				return
			case <-ticker.C:
				clientMgr.FlushRunningSessions()
			}
		}
	}()

	log.Println("[BatchSave] Batch save loop started")
	return cancel
}
