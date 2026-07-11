package updates

import (
	"context"
	"time"

	"core/internal/api"
	cmd "core/utils/shell"

	sdkapi "sdk/api"
)

// StartScheduledUpdateChecker checks for software updates at 2AM local router
// time daily.
func StartScheduledUpdateChecker(g *api.CoreGlobals, scheduler sdkapi.ISchedulerApi) error {
	return scheduler.Cron("scheduled-update-check", "0 2 * * *", func(ctx context.Context) {
		performScheduledUpdateCheck(g)
	})
}

func performScheduledUpdateCheck(g *api.CoreGlobals) {
	// CheckSoftwareReleaseUpdate sources the machine's product version internally
	// (core/product.json, falling back to the core version).
	result, err := CheckSoftwareReleaseUpdate()
	if err != nil || !result.HasUpdate {
		return
	}

	// Force updates must be installed automatically - users cannot opt-out
	if result.ForceUpdate {
		if !IsDownloading() && !IsDownloaded() {
			// startForcedUpdate is build-tag split (scheduler_system.go /
			// scheduler_system_mono.go): a non-mono ordinary core update MUST go
			// through the two-phase staging pipeline (StageSystemUpdate), not the
			// generic single-file download mono uses for everything — see
			// waitForDownloadAndReboot for why conflating the two left this checker
			// unable to ever detect completion for non-mono.
			startForcedUpdate(g, result)
			waitForDownloadAndReboot(result.IsSysupgrade)
		}
		return
	}
}

// waitForDownloadAndReboot waits for the triggered update to finish, then reboots
// (or flashes, for a sysupgrade) the system.
//
// isSysupgrade selects which signal means "done": IsSysupgradeReady() checks the
// downloaded firmware file directly, so it is correct regardless of mono/non-mono.
// IsDownloaded(), by contrast, means something DIFFERENT per build (mono: the
// generic download marker; non-mono: StageSystemUpdate's own staged-complete
// marker — see system-update.go / system-update_mono.go) — using it as the gate
// for a non-mono sysupgrade previously left this loop unable to ever observe
// completion, since FinalizeSysupgrade never touches non-mono's staged-complete
// marker. Gating each case on its own build-tag-correct signal fixes both.
func waitForDownloadAndReboot(isSysupgrade bool) {
	// Poll every 5 seconds until the update completes or fails.
	for {
		time.Sleep(5 * time.Second)

		if DownloadError() != nil {
			return
		}

		// A forced/scheduled update runs unattended — nobody is watching the admin
		// dashboard at 2AM to resolve a plugin-build-failure dialog. Auto-continue
		// (skip the failed plugins, apply the rest) rather than let
		// stageSystemUpdate's confirmation gate block forever: an unresolved gate
		// leaves `downloading` permanently true, which would wedge every future
		// update attempt too, including admin-triggered ones. Safe to call
		// unconditionally (no-op when nothing is waiting).
		if AwaitingPluginConfirm() {
			ResolvePluginFailureDecision(true)
		}

		if isSysupgrade {
			if IsSysupgradeReady() {
				time.Sleep(3 * time.Second)
				// Validate firmware compatibility before flashing
				if err := ValidateSysupgradeCompatibility(); err != nil {
					RemoveSysupgradeFile()
					return
				}
				// Note: Automatic updates always preserve data (noPreserve = false)
				// Only manual uploads can choose to not preserve data
				// In dev mode, shell.Exec will automatically ignore sysupgrade commands
				cmd.Exec(GetSysupgradeCommand(false), nil)
				return
			}
		} else if IsDownloaded() {
			time.Sleep(3 * time.Second)
			// In dev mode, shell.Exec will automatically ignore reboot commands
			cmd.Exec("reboot", nil)
			return
		}

		if !IsDownloading() {
			return
		}
	}
}
