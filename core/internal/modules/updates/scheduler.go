package updates

import (
	"context"
	"time"

	cmd "core/utils/shell"

	sdkapi "sdk/api"
)

// StartScheduledUpdateChecker checks for software updates at 2AM local router
// time daily.
func StartScheduledUpdateChecker(scheduler sdkapi.ISchedulerApi) error {
	return scheduler.Cron("scheduled-update-check", "0 2 * * *", func(ctx context.Context) {
		performScheduledUpdateCheck()
	})
}

func performScheduledUpdateCheck() {
	// CheckSoftwareReleaseUpdate sources the machine's product version internally
	// (core/product.json, falling back to the core version).
	result, err := CheckSoftwareReleaseUpdate()
	if err != nil || !result.HasUpdate {
		return
	}

	// Force updates must be installed automatically - users cannot opt-out
	if result.ForceUpdate {
		if !IsDownloading() && !IsDownloaded() {
			DownloadSoftwareUpdate(DownloadParams{
				FileURL:      result.ReleseFileURL,
				Checksum:     result.ReleaseFileChecksum,
				OutputPath:   GetUpdateOutputPath(result.ReleseFileURL, result.IsSysupgrade),
				IsSysupgrade: result.IsSysupgrade,
			})
			waitForDownloadAndReboot()
		}
		return
	}
}

// waitForDownloadAndReboot waits for the download to complete then reboots the system
func waitForDownloadAndReboot() {
	// Poll every 5 seconds until download completes or fails
	for {
		time.Sleep(5 * time.Second)

		if DownloadError() != nil {
			return
		}

		if IsDownloaded() {
			time.Sleep(3 * time.Second)

			if IsSysupgradeReady() {
				// Validate firmware compatibility before flashing
				if err := ValidateSysupgradeCompatibility(); err != nil {
					RemoveSysupgradeFile()
					return
				}
				// Note: Automatic updates always preserve data (noPreserve = false)
				// Only manual uploads can choose to not preserve data
				// In dev mode, shell.Exec will automatically ignore sysupgrade commands
				cmd.Exec(GetSysupgradeCommand(false), nil)
			} else {
				// In dev mode, shell.Exec will automatically ignore reboot commands
				cmd.Exec("reboot", nil)
			}
			return
		}

		if !IsDownloading() {
			return
		}
	}
}
