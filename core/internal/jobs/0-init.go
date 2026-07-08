package jobs

import (
	"fmt"

	"core/internal/api"
	"core/internal/modules/updates"
	"core/utils/tags"
)

func Init(g *api.CoreGlobals) {
	scheduler := g.CoreAPI.Scheduler()

	// Cloud-dependent schedulers are disabled in devkit builds: the devkit never
	// contacts the cloud (no update check, machine ping, portal cert, denylist, or
	// installed-plugins report). Local jobs below always run. Plugins keep their
	// own networking — only the core's call-home is muted.
	if !tags.IsDevkit() {
		// Start scheduled update checker (runs at 2AM daily)
		logJobErr(g, updates.StartScheduledUpdateChecker(scheduler))

		// Start machine ping scheduler (pings every hour for online status)
		logJobErr(g, StartMachinePingScheduler(scheduler))

		// Start portal certificate scheduler (fetches the shared captive-portal TLS
		// cert from the cloud and hot-reloads HTTPS when it changes)
		logJobErr(g, StartPortalCertScheduler(scheduler))

		// Start blocked-plugins scheduler (polls the cloud denylist once a day and
		// marks offending plugins so the boot loader skips them on the next reboot)
		logJobErr(g, StartBlockedPluginsScheduler(scheduler))

		// Start installed-plugins report scheduler (reports the machine's full set of
		// installed plugins so the cloud can track current installs + install history)
		logJobErr(g, StartInstalledPluginsReportScheduler(scheduler))
	}

	// Start fingerprint cleanup scheduler (runs at 3AM daily)
	logJobErr(g, StartFingerprintCleanupScheduler(scheduler, g.Database, g.Models))

	// Start session cleanup scheduler (runs at 11:30 PM daily, deletes consumed/expired
	// sessions and notifies admin about sessions never started in 90+ days)
	logJobErr(g, StartSessionCleanupScheduler(g.Database, g.Models, g.CoreAPI))

	// Start device log cleanup scheduler (runs every hour, deletes logs older than 90 days)
	logJobErr(g, StartDeviceLogCleanupScheduler(g.Database, g.Models, g.CoreAPI))

	// Start notification cleanup scheduler (runs at 2:00 AM daily, caps the
	// notifications table to its newest rows so it can't grow unbounded)
	logJobErr(g, StartNotificationCleanupScheduler(g.Database, g.Models, g.CoreAPI))

	// Start voucher cleanup scheduler (runs at 2:15 AM daily, deletes stale activated
	// vouchers and notifies admin about vouchers that expired before use)
	logJobErr(g, StartVoucherCleanupScheduler(g.Database, g.Models, g.CoreAPI))

	// Start device-merge reconciliation job (periodically merges duplicate device
	// rows left behind by MAC randomization + lost cookies, when fingerprints
	// confirm they are the same physical device)
	logJobErr(g, StartDeviceMergeScheduler(g.Database, g.Models, g.ClientMgr, g.CoreAPI))

	// Start batch save loop for running sessions (flushes to DB periodically, and
	// once more on graceful shutdown so in-flight usage isn't lost)
	logJobErr(g, StartBatchSaveLoop(scheduler, g.ClientMgr))

	// Start ubus listener for network interface events
	StartUbusListener()

	// Initialize WiFi state tracker and start WiFi event detection
	logJobErr(g, InitAndStartWifiMgr(scheduler, g.WifiMgr))
}

// logJobErr reports a scheduler registration failure. These are static,
// hardcoded, one-time registrations at boot, so an error here means a
// programming mistake (e.g. a duplicate job name or a bad cron expression)
// rather than a runtime condition — but it must still be surfaced rather than
// silently dropped.
func logJobErr(g *api.CoreGlobals, err error) {
	if err != nil {
		g.CoreAPI.Logger().Error(fmt.Sprintf("jobs: %v", err))
	}
}
