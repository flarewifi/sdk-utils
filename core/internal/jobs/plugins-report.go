package jobs

import (
	"context"
	"core/internal/modules/pluginreport"
	"time"

	sdkapi "sdk/api"
)

// StartInstalledPluginsReportScheduler reports the machine's FULL set of installed
// plugins to the cloud shortly after boot and then once a day. The cloud reconciles
// the snapshot into machine_plugins and derives install/update/uninstall history
// from the diff (the ReportInstalledPlugins RPC), so sending the full set rather
// than deltas is intentional and self-healing.
//
// The daily cadence is just a backstop: install/uninstall actions trigger an
// immediate report via pluginreport.ReportNowAsync (see the plugins-manager hooks),
// and the boot report captures anything applied while offline or on the last reboot.
func StartInstalledPluginsReportScheduler(scheduler sdkapi.ISchedulerApi) error {
	return scheduler.Go("installed-plugins-report", func(ctx context.Context) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(InstalledPluginsReportInitialDelay):
		}
		pluginreport.ReportNow()

		ticker := time.NewTicker(InstalledPluginsReportInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pluginreport.ReportNow()
			}
		}
	})
}
