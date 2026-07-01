// Package pluginreport reports the machine's CURRENT set of installed plugins to
// the cloud, which reconciles the snapshot into machine_plugins and derives
// install/update/uninstall history from the diff (the ReportInstalledPlugins v3
// RPC). It is a leaf module imported by both the report scheduler (jobs) and the
// install/uninstall hooks (api) — kept separate so api can trigger a report
// without importing jobs (jobs already imports api, which would be a cycle).
package pluginreport

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v3"
	"core/utils/config"
	"core/utils/crypt"
	"core/utils/plugins"
	"path/filepath"
	"sync"
	"time"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// coalesceWindow batches a burst of nudges (e.g. installing every member of a
// meta bundle fires one per member) into a single report.
const coalesceWindow = 3 * time.Second

var (
	startWorker sync.Once
	trigger     = make(chan struct{}, 1)
)

// ReportNow sends the machine's full installed-plugin snapshot to the cloud
// synchronously. Used by the scheduler (boot + daily). Panics are contained so a
// background caller can never crash the process; a failed send is left for the
// next report (the full snapshot is self-healing).
func ReportNow() {
	defer func() { _ = recover() }()

	_, machineID := machineuid.GetMachineUID()
	if machineID == "" {
		return
	}

	srv, ctx := rpc.GetTwirpServiceAndCtx()
	_, _ = srv.ReportInstalledPlugins(ctx, &rpc_flarewifi_v3.ReportInstalledPluginsRequest{
		MachineId: machineID,
		Plugins:   collect(),
	})
}

// ReportNowAsync requests a report shortly after an install/uninstall completes,
// coalescing bursts into one report. It never blocks the caller. Use this from
// the install/uninstall hooks so a change shows up in the cloud promptly instead
// of waiting for the next daily tick.
func ReportNowAsync() {
	startWorker.Do(func() { go worker() })
	// Non-blocking: a pending trigger already covers this request (the worker
	// re-reads the live snapshot when it runs, so coalescing loses nothing).
	select {
	case trigger <- struct{}{}:
	default:
	}
}

func worker() {
	for range trigger {
		// Wait out the burst window, then drain any extra triggers it produced so
		// the whole burst collapses into this one report.
		time.Sleep(coalesceWindow)
		select {
		case <-trigger:
		default:
		}
		ReportNow()
	}
}

// collect reads every installed plugin's plugin.json (package, name, version) and
// tags it with its install source. A package installed from the store reports
// source "store" (the cloud links it to its plugins row by package); every other
// origin (local, git, system) reports "local". Plugins MARKED for removal are
// excluded so an uninstall (which only takes effect on the next reboot) is
// reflected immediately.
//
// Meta bundles have no install dir, so the plugin-dir scan below can't see them —
// but the cloud needs to know which bundles a machine has installed to resolve
// meta-member coverage (a paid plugin that is free because it belongs to a free or
// purchased bundle) from server state. So the machine's meta RECORDS are reported
// too, as source "store" (a bundle links to its plugins row, is_meta = true, by
// package). Without this the cloud never records the bundle and every one of its
// paid members reads as unpaid.
func collect() []*rpc_flarewifi_v3.InstalledPlugin {
	// package -> install source, from the plugins config (source of truth for how
	// each package was installed). Packages absent here default to "local".
	srcByPkg := make(map[string]string)
	for _, m := range plugins.InstalledPluginsList() {
		srcByPkg[m.Package] = m.Def.Src
	}

	var list []*rpc_flarewifi_v3.InstalledPlugin
	for _, dir := range plugins.InstalledPluginDirs() {
		info, err := sdkutils.GetPluginInfoFromPath(dir)
		if err != nil || info.Package == "" {
			continue
		}
		// A plugin marked to remove is uninstalled from the user's perspective even
		// though its files linger until the next reboot — don't report it as present.
		if plugins.IsToBeRemoved(info.Package) {
			continue
		}

		source := "local"
		if srcByPkg[info.Package] == sdkutils.PluginSrcStore {
			source = "store"
		}

		// Hash the installed plugin.so so the cloud can verify the binary against the
		// build's recorded hash. Absent for statically-linked/system plugins (compiled
		// into core/plugin.so) and meta bundles below — those report an empty hash. A
		// hash failure must not drop the plugin from the snapshot, so ignore the error.
		soHash := ""
		soPath := filepath.Join(dir, "plugin.so")
		if sdkutils.FsExists(soPath) {
			if h, err := crypt.SHA256File(soPath); err == nil {
				soHash = h
			}
		}

		list = append(list, &rpc_flarewifi_v3.InstalledPlugin{
			Package: info.Package,
			Name:    info.Name,
			Version: info.Version,
			Source:  source,
			SoHash:  soHash,
		})
	}

	// Report installed meta bundles from the plugins config (they have no install dir
	// above). A meta record's presence IS its installed state — there is no
	// "marked for removal" for a bundle, so no such filter is needed here.
	if cfg, err := config.ReadPluginsConfig(); err == nil {
		for _, meta := range cfg.MetaPlugins {
			if meta.Package == "" {
				continue
			}
			list = append(list, &rpc_flarewifi_v3.InstalledPlugin{
				Package: meta.Package,
				Name:    meta.Name,
				Version: meta.Version,
				Source:  sdkutils.PluginSrcStore,
			})
		}
	}
	return list
}
