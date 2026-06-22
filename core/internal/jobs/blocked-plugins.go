package jobs

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v2"
	"core/utils/plugins"
	"time"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// StartBlockedPluginsScheduler polls the cloud denylist once a day and reconciles
// the on-disk "blocked" markers so the boot loader skips offending plugins. The
// initial fetch runs shortly after boot so a freshly-flagged plugin is caught on
// the next reboot without waiting a full day.
func StartBlockedPluginsScheduler() {
	go func() {
		time.Sleep(BlockedPluginsInitialDelay)
		reconcileBlockedPlugins()

		ticker := time.NewTicker(BlockedPluginsInterval)
		defer ticker.Stop()
		for range ticker.C {
			reconcileBlockedPlugins()
		}
	}()
}

func reconcileBlockedPlugins() {
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
	resp, err := srv.FetchBlockedPlugins(ctx, &rpc_flarewifi_v2.FetchBlockedPluginsRequest{
		MachineId: machineID,
	})
	if err != nil {
		// Network/cloud hiccup: leave existing markers untouched and retry next
		// tick. We never clear blocks on a failed fetch — that would unblock an
		// offending plugin just because the machine briefly lost connectivity.
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
			_ = plugins.BlockPlugin(info.Package)
		case !shouldBlock && plugins.IsBlocked(info.Package):
			_ = plugins.UnblockPlugin(info.Package)
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
