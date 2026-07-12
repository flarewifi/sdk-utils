package jobs

import (
	"context"
	"fmt"

	"core/internal/api"
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v3"
	"core/utils/plugins"
	"time"

	sdkutils "github.com/flarewifi/sdk-utils"
	sdkapi "sdk/api"
)

// StartBlockedPluginsScheduler polls the cloud denylist once a day and reconciles
// the on-disk "blocked" markers so the boot loader skips offending plugins. The
// initial fetch runs shortly after boot so a freshly-flagged plugin is caught on
// the next reboot without waiting a full day.
//
// g is threaded through (rather than just the raw *scheduler.Manager) so
// reconcileBlockedPlugins can both cancel a newly-blocked plugin's OWN
// scheduled tasks immediately via CancelOwner, rather than waiting for the
// next reboot, and log when an unblock can't fully undo that cancellation
// (see the CancelOwner comment below).
//
// reconcileBlockedPlugins recovers its own panics (see below) specifically so
// the ticker loop survives a bad tick and keeps retrying daily; that resilience
// would be lost by registering it via Every/Cron instead, since the scheduler
// Manager stops a periodic task for good after one panic.
func StartBlockedPluginsScheduler(sched sdkapi.ISchedulerApi, g *api.CoreGlobals) error {
	return sched.Go("blocked-plugins", func(ctx context.Context) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(BlockedPluginsInitialDelay):
		}
		reconcileBlockedPlugins(g)

		ticker := time.NewTicker(BlockedPluginsInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				reconcileBlockedPlugins(g)
			}
		}
	})
}

func reconcileBlockedPlugins(g *api.CoreGlobals) {
	// This runs in a background goroutine, never on the boot path — but a panic in
	// a goroutine crashes the whole process, so contain it here. The shared RPC
	// helper (rpc.GetTwirpServiceAndCtx) panics on a header error, and an offending
	// denylist must never be able to brick a machine or abort its boot. Recovering
	// also keeps the ticker loop alive to retry on the next tick.
	defer func() { _ = recover() }()

	_, machineID := machineuid.GetMachineUID()
	if machineID == "" {
		return
	}

	srv, ctx := rpc.GetTwirpServiceAndCtx()
	resp, err := srv.FetchBlockedPlugins(ctx, &rpc_flarewifi_v3.FetchBlockedPluginsRequest{
		MachineId: machineID,
	})
	if err != nil {
		// Network/cloud hiccup: leave existing markers untouched and retry next
		// tick. We never clear blocks on a failed fetch — that would unblock an
		// offending plugin just because the machine briefly lost connectivity.
		return
	}
	if !resp.GetSuccess() {
		// The RPC always answers HTTP 200; Success=false means the server could
		// not compute a definitive denylist (e.g. a DB hiccup), NOT that the
		// denylist is empty. Treat it the same as a transport failure — leave
		// every marker untouched rather than reading the (unpopulated) lists
		// below as "nothing is blocked".
		return
	}

	blockedPackages := sliceToSet(resp.GetBlockedPackages())
	blockedNames := sliceToSet(resp.GetBlockedNames())

	// Reconcile every installed plugin (store, local, devel, system) against the
	// denylist: matching plugins get a block marker, plugins that fell off the
	// list get theirs cleared. Errors on one plugin must not abort the others.
	for _, dir := range plugins.InstalledPluginDirs() {
		info, err := sdkutils.GetPluginInfoFromPath(dir)
		if err != nil {
			continue
		}

		shouldBlock := blockedPackages[info.Package] || blockedNames[info.Name]

		switch {
		case shouldBlock && !plugins.IsBlocked(info.Package):
			if err := plugins.BlockPlugin(info.Package); err == nil {
				// The plugin's HTTP routes are already gated per-request by
				// middlewares.PluginValidityCheck; this stops its background
				// work too, immediately, instead of leaving it running until
				// the next reboot.
				g.SchedulerMgr.CancelOwner(info.Package)
			}
		case !shouldBlock && plugins.IsBlocked(info.Package):
			if err := plugins.UnblockPlugin(info.Package); err == nil && g.SchedulerMgr.HadCancelledTasks(info.Package) {
				// Its background Go/Every/Cron tasks were killed by the earlier
				// BlockPlugin+CancelOwner and cannot be re-registered without
				// re-running the plugin's Init, which only ever runs once at
				// boot. Surface this instead of leaving it silently
				// half-working now that its HTTP routes resume immediately.
				g.CoreAPI.Logger().Error(fmt.Sprintf("plugin %q unblocked, but its background tasks were stopped while blocked and require a machine reboot to resume", info.Package))
			}
		}
	}
}

func sliceToSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}
