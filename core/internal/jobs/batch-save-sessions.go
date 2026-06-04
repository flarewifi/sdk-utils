package jobs

import (
	"context"
	"time"

	"core/internal/sessmgr"
)

func StartBatchSaveLoop(clientMgr *sessmgr.SessionsMgr) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
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
	}()

	return cancel
}
