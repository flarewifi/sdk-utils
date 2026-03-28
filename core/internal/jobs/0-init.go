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

	// Start device merge scheduler — merges duplicate devices that share a historical
	// MAC address and have matching full browser fingerprints (CNA-only skipped).
	StartDeviceMergeScheduler(g)

	// Start log cleanup scheduler (runs at 4AM daily, deletes logs older than 7 days)
	StartLogCleanupScheduler(g.Database, g.Models)

	// Start machine ping scheduler (pings every hour for online status)
	StartMachinePingScheduler()

	// Start batch save loop for running sessions (flushes to DB periodically)
	StartBatchSaveLoop(g.ClientMgr)

	// Start ubus listener for network interface events
	StartUbusListener()

	// Initialize WiFi state tracker and start WiFi event detection
	InitAndStartWifiMgr(g.WifiMgr)
}
