package updates

import (
	"log"
	"time"

	cmd "core/utils/shell"

	"github.com/Masterminds/semver/v3"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

// StartScheduledUpdateChecker starts a background goroutine that checks for
// software updates at 2AM local router time daily
func StartScheduledUpdateChecker() {
	go func() {
		for {
			// Calculate duration until next 2AM
			now := time.Now()
			next2AM := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
			if now.After(next2AM) {
				next2AM = next2AM.Add(24 * time.Hour)
			}

			time.Sleep(next2AM.Sub(now))
			performScheduledUpdateCheck()
		}
	}()
}

func performScheduledUpdateCheck() {
	coreInfo, err := sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir)
	if err != nil {
		return
	}

	currentVersion, err := semver.NewVersion(coreInfo.Version)
	if err != nil {
		return
	}

	result, err := CheckSoftwareReleaseUpdate(currentVersion)
	if err != nil || !result.HasUpdate {
		return
	}

	// Force updates must be installed automatically - users cannot opt-out
	if result.ForceUpdate {
		log.Println("Force update detected - will be installed automatically")
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
			log.Println("Automatic software update downloaded")
			time.Sleep(3 * time.Second)

			if IsSysupgradeReady() {
				// Validate firmware compatibility before flashing
				if err := ValidateSysupgradeCompatibility(); err != nil {
					log.Println("Sysupgrade compatibility check failed, aborting auto-update:", err)
					RemoveSysupgradeFile()
					return
				}
				log.Println("Sysupgrade compatibility check passed - flashing firmware now")
				// Note: Automatic updates always preserve data (noPreserve = false)
				// Only manual uploads can choose to not preserve data
				// In dev mode, shell.Exec will automatically ignore sysupgrade commands
				cmd.Exec(GetSysupgradeCommand(false), nil)
			} else {
				log.Println("Rebooting to apply software update")
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
