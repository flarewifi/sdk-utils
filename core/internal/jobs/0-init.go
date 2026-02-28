package jobs

import (
	"log"

	"core/internal/api"
	"core/internal/modules/updates"
)

func Init(g *api.CoreGlobals) {
	log.Println("[Jobs] Initializing background jobs...")

	// Start scheduled update checker (runs at 2AM daily)
	log.Println("[Jobs] Starting scheduled update checker...")
	updates.StartScheduledUpdateChecker()

	// Start fingerprint cleanup scheduler (runs at 3AM daily)
	log.Println("[Jobs] Starting fingerprint cleanup scheduler...")
	StartFingerprintCleanupScheduler(g.Database, g.Models)

	// Start log cleanup scheduler (runs at 4AM daily, deletes logs older than 7 days)
	log.Println("[Jobs] Starting log cleanup scheduler...")
	StartLogCleanupScheduler(g.Database, g.Models)

	// Start ubus listener for network interface events
	log.Println("[Jobs] Starting ubus listener...")
	StartUbusListener()

	// Start WiFi event listener to bridge WifiMgr events to legacy callbacks
	log.Println("[Jobs] Starting WiFi event listener bridge...")
	StartWifiEventListener(g.WifiMgr)

	log.Println("[Jobs] All background jobs initialized")
}
