// Package plugindeps fetches the cloud's per-core-version dependency lock so a
// plugin compiled ON THIS MACHINE can be pinned to the exact module versions+hashes
// the core and every other plugin were built against. A Go plugin.so only loads when
// all shared packages were compiled byte-identically; the cloud builder already
// enforces this for server-side builds, and this lets local builds match it.
package plugindeps

import (
	"fmt"

	machineuid "core/internal/modules/machine-uid"
	corerpc "core/internal/rpc"
	v2 "core/internal/rpc/rpc_flarewifi_v2"
	"core/utils/plugins"
)

// Fetch returns the dependency lock for coreVersion (empty = the machine's
// registered core version; pass the TARGET version when staging a core update).
//
// It degrades gracefully: any RPC failure (offline machine, unregistered machine,
// transient error) or an empty lock (first plugin for a core version, or a core
// version whose lock has not been seeded) returns nil and logs a warning. Callers
// then build with no pins — the same behavior as before this fetch existed — so a
// local install never fails just because the lock is unavailable.
func Fetch(coreVersion string) []plugins.LockedGoModule {
	srv, ctx := corerpc.GetTwirpServiceAndCtx()
	_, machineID := machineuid.GetMachineUID()

	resp, err := srv.FetchPluginDependencies(ctx, &v2.FetchPluginDependenciesRequest{
		MachineId:   machineID,
		CoreVersion: coreVersion,
	})
	if err != nil {
		// Best-effort: log only that the dependency-lock fetch failed. The raw RPC
		// error carries the cloud endpoint URL/domain, which must never reach logs.
		fmt.Printf("[plugindeps] warning: unable to fetch the plugin dependency lock from the cloud (building unpinned)\n")
		return nil
	}

	modules := resp.GetModules()
	if len(modules) == 0 {
		// No lock recorded for this core version yet — build unpinned (this build may
		// be the first to establish the set the cloud later folds into the lock).
		return nil
	}

	locked := make([]plugins.LockedGoModule, 0, len(modules))
	for _, m := range modules {
		locked = append(locked, plugins.LockedGoModule{
			Path:      m.GetPath(),
			Version:   m.GetVersion(),
			Hash:      m.GetHash(),
			GoModHash: m.GetGoModHash(),
		})
	}
	fmt.Printf("[plugindeps] pinned %d module(s) from the core dependency lock (core_version=%q)\n", len(locked), coreVersion)
	return locked
}
