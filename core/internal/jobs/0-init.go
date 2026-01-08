package jobs

import (
	"core/internal/api"
	"core/internal/modules/updates"
)

func Init(g *api.CoreGlobals) {

	// Start scheduled update checker (runs at 2AM daily)
	updates.StartScheduledUpdateChecker()

	// Start fingerprint cleanup scheduler (runs at 3AM daily)
	StartFingerprintCleanupScheduler(g.Database, g.Models)
}
