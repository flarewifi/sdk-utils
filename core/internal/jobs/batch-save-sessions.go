package jobs

import (
	"context"
	"time"

	"core/internal/sessmgr"

	sdkapi "sdk/api"
)

// StartBatchSaveLoop periodically flushes running sessions' accumulated usage
// to the database, and once more on graceful shutdown so in-flight usage
// isn't lost. Registering via the scheduler (rather than a bare goroutine plus
// a returned context.CancelFunc) means shutdown is handled automatically —
// previously the returned CancelFunc was silently discarded by the caller.
func StartBatchSaveLoop(scheduler sdkapi.ISchedulerApi, clientMgr *sessmgr.SessionsMgr) error {
	return scheduler.Go("batch-save-sessions", func(ctx context.Context) {
		ticker := time.NewTicker(BatchSaveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				clientMgr.FlushRunningSessions()
				return
			case <-ticker.C:
				clientMgr.FlushRunningSessions()
			}
		}
	})
}
