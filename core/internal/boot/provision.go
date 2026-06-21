package boot

import (
	"context"
	"fmt"
	"sync/atomic"

	"core/internal/api"
	"core/internal/modules/netmon"
	"core/utils/plugins"

	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// provisioning guards against overlapping provisioning passes: connectivity can
// flap up→down→up while a pass is still running opkg/pip, and we must not start a
// second pass on top of the first.
var provisioning atomic.Bool

// StartOnlineMonitor wires the core's online monitor to network-dependent install
// work and starts it. On every internet-up transition (including the first probe
// at boot, if the device is already online) it runs a provisioning pass that
// installs each loaded plugin's system_packages and runs its preinstall and
// postinstall scripts — the steps that need the network and therefore cannot run
// reliably during the offline part of boot.
//
// Plugins can subscribe to the same EventInternetUp / EventInternetDown signals
// via api.Events().OnInternetEvent.
func StartOnlineMonitor(g *api.CoreGlobals) {
	g.EventsMgr.OnInternetEvent(sdkapi.EventInternetUp, func(ctx context.Context) error {
		// Spawn our own goroutine: provisioning runs opkg/pip and can take
		// minutes, and the events contract requires slow handlers not to block
		// the monitor's polling loop. The atomic guard collapses repeated
		// up-transitions into a single in-flight pass.
		if !provisioning.CompareAndSwap(false, true) {
			return nil
		}
		go func() {
			defer provisioning.Store(false)
			ProvisionInstalledPlugins(g)
		}()
		return nil
	})

	monitor := netmon.NewMonitor(g.EventsMgr, g.CoreAPI.Logger())
	monitor.Start(context.Background())
}

// InitLoadedPlugins runs Init for every loaded plugin whose internet-dependent
// install steps are already satisfied — i.e. it has no unprovisioned
// system_packages or preinstall (see needsProvision). These plugins are safe to
// initialize at boot, even offline, so the device's core function (e.g. the
// captive portal) comes up without waiting for the network.
//
// Plugins that DO need provisioning are skipped here; their Init runs later, in
// ProvisionInstalledPlugins, after their system_packages/preinstall succeed. The
// returned error is the first Init failure (boot treats it like InitPlugins).
func InitLoadedPlugins(g *api.CoreGlobals) error {
	var firstErr error
	for _, p := range g.PluginMgr.PluginApis() {
		info := p.Info()
		if needsProvision(info) {
			// Deferred to the internet-up provisioning pass.
			continue
		}
		if err := p.RunInit(); err != nil {
			if logErr := g.CoreAPI.Logger().Error(fmt.Sprintf("plugin %q Init failed: %v", info.Package, err)); logErr != nil {
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("plugin %q: %w", info.Package, err)
			}
		}
	}
	return firstErr
}

// ProvisionInstalledPlugins runs the network-dependent install steps for every
// loaded plugin, each guarded by a version-pinned marker so it runs once per
// plugin version and is retried on the next internet-up if it failed. Per plugin,
// in order:
//
//  1. system_packages via opkg (ProvisionSystemPkgs, "syspkgs" marker)
//  2. the preinstall script ("preinstall" marker)
//  3. Init — invoked here for plugins whose Init was deferred at boot, so it runs
//     AFTER preinstall (and only once; InitDone guards against a second run)
//  4. the postinstall script ("postinstall" marker), which runs after Init
//
// It iterates loaded plugins (not on-disk dirs) so a plugin that failed to load —
// and was not recovered — is not provisioned. Failures are logged, never fatal.
func ProvisionInstalledPlugins(g *api.CoreGlobals) {
	for _, p := range g.PluginMgr.PluginApis() {
		info := p.Info()
		dir := p.Dir()

		if err := plugins.ProvisionSystemPkgs(info); err != nil {
			if logErr := g.CoreAPI.Logger().Error(fmt.Sprintf("plugin %q system_packages install failed: %v", info.Package, err)); logErr != nil {
			}
			// Leave the marker unwritten so the next internet-up retries; still
			// attempt the scripts, which may not depend on the packages.
		}

		runInstallScriptOnce(g, dir, info, info.PreInstall, "preinstall")

		// Init now that preinstall has run. No-op if it already ran at boot
		// (a non-deferred plugin) — InitDone guards the single run.
		if !p.InitDone() {
			if err := p.RunInit(); err != nil {
				if logErr := g.CoreAPI.Logger().Error(fmt.Sprintf("plugin %q Init failed during provisioning: %v", info.Package, err)); logErr != nil {
				}
				// Skip postinstall when Init failed — postinstall runs after a
				// successful Init. A later internet-up retries the whole pass.
				continue
			}
		}

		runInstallScriptOnce(g, dir, info, info.PostInstall, "postinstall")
	}
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// needsProvision reports whether a plugin has internet-dependent install work
// (system_packages or a preinstall script) that has not yet completed for its
// installed version. Such a plugin's Init is deferred until the online monitor's
// provisioning pass runs that work; a plugin with nothing pending initializes at
// boot. Markers make this self-resetting on upgrade (a new version invalidates
// the marker) and skip re-work once a version is fully provisioned.
func needsProvision(info sdkutils.PluginInfo) bool {
	if len(info.SystemPackages) > 0 && plugins.ReadScriptMarker(info.Package, "syspkgs") != info.Version {
		return true
	}
	if info.PreInstall != "" && plugins.ReadScriptMarker(info.Package, "preinstall") != info.Version {
		return true
	}
	return false
}
