package jobs

import (
	"core/internal/api"
	"core/internal/modules/updates"
	"core/utils/tags"
)

func Init(g *api.CoreGlobals) {
	// Cloud-dependent schedulers are disabled in devkit builds: the devkit never
	// contacts the cloud (no update check, machine ping, portal cert, denylist, or
	// installed-plugins report). Local jobs below always run. Plugins keep their
	// own networking — only the core's call-home is muted.
	if !tags.IsDevkit() {
		// Start scheduled update checker (runs at 2AM daily)
		updates.StartScheduledUpdateChecker()

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
	}

	// Start fingerprint cleanup scheduler (runs at 3AM daily)
	StartFingerprintCleanupScheduler(g.Database, g.Models)

	// Start session cleanup scheduler (runs at 11:30 PM daily, deletes consumed/expired
	// sessions and notifies admin about sessions never started in 90+ days)
	StartSessionCleanupScheduler(g.Database, g.Models, g.CoreAPI)

	// Start device log cleanup scheduler (runs every hour, deletes logs older than 90 days)
	StartDeviceLogCleanupScheduler(g.Database, g.Models, g.CoreAPI)

	// Start notification cleanup scheduler (runs at 2:00 AM daily, caps the
	// notifications table to its newest rows so it can't grow unbounded)
	StartNotificationCleanupScheduler(g.Database, g.Models, g.CoreAPI)

	// Start voucher cleanup scheduler (runs at 2:15 AM daily, deletes stale activated
	// vouchers and notifies admin about vouchers that expired before use)
	StartVoucherCleanupScheduler(g.Database, g.Models, g.CoreAPI)

	// Start device-merge reconciliation job (periodically merges duplicate device
	// rows left behind by MAC randomization + lost cookies, when fingerprints
	// confirm they are the same physical device)
	StartDeviceMergeScheduler(g.Database, g.Models, g.ClientMgr, g.CoreAPI)

	// Start batch save loop for running sessions (flushes to DB periodically)
	StartBatchSaveLoop(g.ClientMgr)

	// Start ubus listener for network interface events
	StartUbusListener()

	// Initialize WiFi state tracker and start WiFi event detection
	InitAndStartWifiMgr(g.WifiMgr)
}
