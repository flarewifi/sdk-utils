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

	// Start log cleanup scheduler (runs at 4AM daily, deletes logs older than 7 days)
	StartLogCleanupScheduler(g.Database, g.Models)

	// Start machine ping scheduler (pings every hour for online status)
	StartMachinePingScheduler()

	// Start ubus listener for network interface events
	StartUbusListener()

	// Initialize WiFi state tracker and start WiFi event detection
	InitAndStartWifiMgr(g.WifiMgr)
}
