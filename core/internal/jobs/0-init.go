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

	// Start session cleanup scheduler (runs at 11:30 PM daily, deletes consumed/expired sessions)
	StartSessionCleanupScheduler(g.Database, g.Models)

	// Start machine ping scheduler (pings every hour for online status)
	StartMachinePingScheduler()

	// Start portal certificate scheduler (fetches the shared captive-portal TLS
	// cert from the cloud and hot-reloads HTTPS when it changes)
	StartPortalCertScheduler()

	// Start blocked-plugins scheduler (polls the cloud denylist once a day and
	// marks offending plugins so the boot loader skips them on the next reboot)
	StartBlockedPluginsScheduler()

	// Start installed-plugins report scheduler (reports the machine's full set of
	// installed plugins so the cloud can track current installs + install history)
	StartInstalledPluginsReportScheduler()

	// Start batch save loop for running sessions (flushes to DB periodically)
	StartBatchSaveLoop(g.ClientMgr)

	// Start ubus listener for network interface events
	StartUbusListener()

	// Initialize WiFi state tracker and start WiFi event detection
	InitAndStartWifiMgr(g.WifiMgr)
}
