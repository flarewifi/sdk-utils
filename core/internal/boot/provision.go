package boot

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"core/internal/api"
	"core/internal/modules/activation"
	"core/internal/modules/bootprogress"
	"core/internal/modules/netmon"
	"core/utils/env"
	"core/utils/plugins"

	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// provisioning guards against overlapping provisioning passes: connectivity can
// flap up→down→up while a pass is still running opkg/pip, and we must not start a
// second pass on top of the first.
var provisioning atomic.Bool

// validatingStore guards against overlapping store-plugin purchase validation
// passes for the same reason connectivity can flap mid-pass.
var validatingStore atomic.Bool

// activating guards against overlapping cloud activation/registration passes: the
// boot-time kick and the online monitor's per-reconnect kick must not run
// activation.Validate() concurrently (it writes the activation token file).
var activating atomic.Bool

// StartActivation runs one cloud registration/activation pass in its own goroutine,
// behind the `activating` guard so overlapping calls collapse into the in-flight
// pass. Safe to call repeatedly. Registration is decoupled from provisioning on
// purpose: the machine must appear in the fleet even when on-device plugin install
// stalls or OOMs the process. Returns immediately; the pass runs in the background.
func StartActivation() {
	if !activating.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer activating.Store(false)
		activation.Validate()
	}()
}

// StartOnlineMonitor wires the core's online monitor to network-dependent install
// work and arranges for it to start once boot completes (it subscribes to EventBoot
// rather than starting the monitor inline — see the bottom of this function). On
// every internet-up transition (including the monitor's first probe, if the device
// is already online when boot finishes) it runs a provisioning pass that installs
// each loaded plugin's system_packages and runs its preinstall and postinstall
// scripts — the steps that need the network and therefore cannot run reliably
// during the offline part of boot.
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
			// Reconnect-time pass: no boot page is showing, so no progress tracker.
			ProvisionInstalledPlugins(g, nil)
		}()
		return nil
	})

	// Re-validate installed store plugins' purchases each time the machine is
	// online (immediately at boot if already online). A separate guard + goroutine
	// from provisioning so the two run concurrently and neither blocks the monitor.
	g.EventsMgr.OnInternetEvent(sdkapi.EventInternetUp, func(ctx context.Context) error {
		if !validatingStore.CompareAndSwap(false, true) {
			return nil
		}
		go func() {
			defer validatingStore.Store(false)
			ValidateStorePlugins(g)
		}()
		return nil
	})

	// Register/re-validate activation with the cloud each time the machine is online
	// (immediately at boot if already online, and on every reconnect). The boot
	// sequence already kicks this once before provisioning; re-running it here makes
	// registration self-healing for a machine that was offline at boot or whose first
	// attempt failed — without waiting for a reboot. Guarded + own goroutine like the
	// passes above, so it runs concurrently and never blocks the monitor.
	g.EventsMgr.OnInternetEvent(sdkapi.EventInternetUp, func(ctx context.Context) error {
		StartActivation()
		return nil
	})

	// Tell the admin (admin notifications only — never the captive portal) when the
	// machine loses internet, so they know network setup may be needed. Spawned so a
	// slow notification write can't stall the monitor's polling loop.
	g.EventsMgr.OnInternetEvent(sdkapi.EventInternetDown, func(ctx context.Context) error {
		go notifyOffline(g)
		return nil
	})

	// Defer the monitor's polling loop until boot completes (EventBoot). During boot
	// the WAN link is still coming up (DHCP/PPPoE/default route), so probing then can
	// emit a spurious EventInternetDown — a false "No internet connection" admin
	// notification on every reboot — and run the internet-up provisioning/activation
	// passes while boot is still finalizing. Starting after boot:complete gates both.
	//
	// Capture context.Background() for the monitor's lifetime: the ctx handed to a
	// boot callback is cancelled the moment the callback returns, so retaining it
	// would stop the polling loop almost immediately.
	monitor := netmon.NewMonitor(g.EventsMgr, g.CoreAPI.Logger())
	g.EventsMgr.OnBootEvent(sdkapi.EventBoot, func(ctx context.Context) error {
		monitor.Start(context.Background())
		return nil
	})
}

// provisionBootCap bounds how long boot will BLOCK on the first provisioning pass
// before proceeding regardless. Provisioning (opkg/pip + the deferred plugins'
// Init) is shown on the booting page, but it must never hang boot: opkg/pip on a
// flaky link can stall for minutes, and the captive portal has to come up. If the
// pass exceeds this cap, boot continues offline-first and the pass keeps running
// in the background under the provisioning guard (the online monitor also retries
// on the next internet-up). The cap is generous so a normal install still gates.
const provisionBootCap = 5 * time.Minute

// RunBootProvisioning runs the boot-time provisioning pass bounded by
// provisionBootCap, so boot always completes even if opkg/pip or a deferred
// plugin's Init blocks. It holds the shared `provisioning` guard for the whole
// pass (even past the cap, while it finishes in the background) so the online
// monitor's internet-up handler cannot start a second, overlapping pass.
func RunBootProvisioning(g *api.CoreGlobals) {
	// Should always succeed here (the monitor isn't started yet), but honor the
	// guard for safety; if a pass is somehow already in flight, don't start another.
	if !provisioning.CompareAndSwap(false, true) {
		return
	}

	done := make(chan struct{})
	go func() {
		defer provisioning.Store(false)
		defer close(done)
		ProvisionInstalledPlugins(g, g.BootProgress)
	}()

	select {
	case <-done:
		// Provisioning finished within the cap — the gated, common case.
	case <-time.After(provisionBootCap):
		// Taking too long: proceed with boot so the captive portal comes up. The
		// goroutine keeps running under the guard and the monitor will retry later.
		g.CoreAPI.Logger().Info("boot: provisioning exceeded the boot cap; continuing and finishing it in the background")
	}
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
		// Show each plugin on the booting page as its Init runs — but only when the
		// loader didn't already publish the per-plugin checklist during load (mono;
		// see loaderEmitsPluginProgress). For non-mono the load loop owns the
		// checklist, so re-emitting here would list every plugin twice.
		if !loaderEmitsPluginProgress {
			g.BootProgress.Substep(pluginDisplayName(info))
		}
		if err := p.RunInit(); err != nil {
			g.CoreAPI.Logger().Error(fmt.Sprintf("plugin %q Init failed: %v", info.Package, err))
			if firstErr == nil {
				firstErr = fmt.Errorf("plugin %q: %w", info.Package, err)
			}
		}
	}
	return firstErr
}

// pluginDisplayName is the human-facing label for a plugin on the booting page:
// its declared Name, falling back to the package id when a plugin omits a name.
func pluginDisplayName(info sdkutils.PluginInfo) string {
	if info.Name != "" {
		return info.Name
	}
	return info.Package
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
//
// progress may be nil (the online monitor's reconnect pass, with no boot page
// showing). When set (the boot-time pass), the current active step's count is
// updated as each plugin that still needs network-dependent work is processed, so
// the booting page can show "installing packages, plugin N of M".
func ProvisionInstalledPlugins(g *api.CoreGlobals, progress *bootprogress.Tracker) {
	apis := g.PluginMgr.PluginApis()

	// Count plugins with pending network-dependent work up front for the progress
	// denominator — evaluated before the loop, since each ProvisionSystemPkgs /
	// install-script call writes a marker that flips needsProvision to false.
	total := 0
	if progress != nil {
		for _, p := range apis {
			if needsProvision(p.Info()) {
				total++
			}
		}
	}

	processed := 0
	for _, p := range apis {
		info := p.Info()
		dir := p.Dir()

		if progress != nil && needsProvision(info) {
			processed++
			progress.SetActiveProgress(processed, total)
		}

		if err := plugins.ProvisionSystemPkgs(info); err != nil {
			// ProvisionSystemPkgs already retried the opkg work several times
			// (see InstallSystemPkgs); reaching here means it still failed, so
			// flag it for the operator.
			notifySystemPkgsFailure(g, info.Package, err)
			// Leave the marker unwritten so the next internet-up retries; still
			// attempt the scripts, which may not depend on the packages.
		}

		runInstallScriptOnce(g, dir, info, info.PreInstall, "preinstall")

		// Init now that preinstall has run. No-op if it already ran at boot
		// (a non-deferred plugin) — InitDone guards the single run.
		if !p.InitDone() {
			if err := p.RunInit(); err != nil {
				g.CoreAPI.Logger().Error(fmt.Sprintf("plugin %q Init failed during provisioning: %v", info.Package, err))
				// Skip postinstall when Init failed — postinstall runs after a
				// successful Init. A later internet-up retries the whole pass.
				continue
			}
		}

		runInstallScriptOnce(g, dir, info, info.PostInstall, "postinstall")
	}
}

// ValidateStorePlugins re-checks every installed STORE plugin with the cloud now
// that the machine is online and withholds any that can no longer run here. Two
// verdicts disable a plugin, checked in this order:
//   - UNAVAILABLE: the cloud reports the plugin as withdrawn/disabled by its
//     developer (check.Available == false) — it can never be installed or loaded,
//     whatever its price. Disabled regardless of free/paid.
//   - PAYMENT LAPSED: a paid plugin this machine is no longer purchased to
//     (subscription expired, refunded, or dropped from a meta bundle).
//
// A withheld plugin is DISABLED — its files are kept on disk but it is skipped by
// the boot loader on the next boot (a loaded Go .so cannot be unmapped at runtime)
// — and the admin is notified (with the matching reason). Each cloud check is
// retried (boot-time internet is often unstable); only a definitive verdict
// disables, so a transient outage never withholds a working plugin. A plugin that
// comes back available AND purchased has any stale disabled marker cleared so it
// loads again next boot.
//
// Local/system/devel plugins are not store plugins and are skipped.
func ValidateStorePlugins(g *api.CoreGlobals) {
	for _, dir := range plugins.InstalledPluginDirs() {
		info, err := sdkutils.GetPluginInfoFromPath(dir)
		if err != nil {
			continue
		}
		def, err := plugins.GetPluginDef(info.Package)
		if err != nil || def.Src != sdkutils.PluginSrcStore {
			continue
		}
		pkg := info.Package

		// The store package is what the cloud keys purchases on (== the plugin's
		// own package for store plugins). Retry to ride out unstable boot internet.
		check, err := sdkutils.Retry(func() (sdkapi.PluginPurchaseInfo, error) {
			return g.PluginMgr.CheckPurchase(def.StorePackage)
		}, 5)
		if err != nil {
			// Still couldn't reach the cloud after retries — leave the plugin as it
			// is (never withhold on a connectivity failure). A later internet-up
			// retries the whole pass.
			g.CoreAPI.Logger().Error(fmt.Sprintf("validate store plugin %q purchase: %v", pkg, err))
			continue
		}

		// A plugin the cloud reports as unavailable (currently: withdrawn/disabled by
		// its developer) can never be installed or loaded, whatever its price. Gate it
		// BEFORE the payment check — mirroring the install/resolve order — so a withdrawn
		// plugin is disabled regardless of free/paid, and the admin sees a "no longer
		// available" reason instead of a misleading "purchase required". A paid but
		// unpurchased plugin stays Available and falls through to the payment branch.
		if !check.Available {
			alreadyDisabled := plugins.IsDisabled(pkg)
			if err := plugins.DisablePlugin(pkg); err != nil {
				g.CoreAPI.Logger().Error(fmt.Sprintf("disable unavailable store plugin %q: %v", pkg, err))
				continue
			}
			if !alreadyDisabled {
				notifyPluginUnavailable(g, pkg)
			}
			continue
		}

		if check.RequiresPayment() {
			alreadyDisabled := plugins.IsDisabled(pkg)
			if err := plugins.DisablePlugin(pkg); err != nil {
				g.CoreAPI.Logger().Error(fmt.Sprintf("disable unpaid store plugin %q: %v", pkg, err))
				continue
			}
			// Notify only on the transition into the disabled state, so a reconnect
			// (which re-runs this pass) doesn't re-notify for an already-withheld plugin.
			if !alreadyDisabled {
				notifyPluginPaymentRequired(g, pkg)
			}
			continue
		}

		// Purchase confirmed (paid, free, or meta-covered): clear any prior disabled
		// marker so the loader picks the plugin up again on the next boot.
		if plugins.IsDisabled(pkg) {
			if err := plugins.EnablePlugin(pkg); err != nil {
				g.CoreAPI.Logger().Error(fmt.Sprintf("re-enable store plugin %q after purchase confirmed: %v", pkg, err))
			}
		}
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

// NeedsProvisionAny reports whether any loaded plugin still has network-dependent
// install work pending for its installed version. Boot uses it to decide whether
// to hold the booting page through the online-wait + system_packages phases (when
// true) or come straight up offline-first (when false — the common case once a
// version is fully provisioned).
func NeedsProvisionAny(g *api.CoreGlobals) bool {
	for _, p := range g.PluginMgr.PluginApis() {
		if needsProvision(p.Info()) {
			return true
		}
	}
	return false
}

// notifySystemPkgsFailure records that a plugin's system_packages could not be
// installed even after InstallSystemPkgs exhausted its retries — typically the
// machine's internet is still unstable. The detailed cause is always logged; in
// production it additionally raises an admin notification so the operator knows
// the plugin may be degraded until the next internet-up pass succeeds. The
// user-facing text names only the plugin package, never the underlying opkg
// error, per the error-message hygiene rules.
func notifySystemPkgsFailure(g *api.CoreGlobals, pkg string, cause error) {
	if err := g.CoreAPI.Logger().Error(fmt.Sprintf("plugin %q system_packages install failed after retries: %v", pkg, cause)); err != nil {
	}

	// Only dev stays silent (failures show in the console); every deployed device
	// (staging, sandbox, production) notifies the operator that a plugin may be
	// degraded until the next internet-up pass.
	if env.IsDevEnv() {
		return
	}

	subject := g.CoreAPI.Translate("error", "Plugin packages failed to install")
	content := fmt.Sprintf("%s: %s", g.CoreAPI.Translate("error", "Required packages for a plugin could not be installed, possibly due to an unstable internet connection. It will be retried automatically when the machine reconnects"), pkg)

	if err := g.CoreAPI.Notification().AddNotification(context.Background(), sdkapi.AddNotificationParams{
		Subject: subject,
		Content: content,
		Type:    sdkapi.NotificationTypeError,
	}); err != nil {
		g.CoreAPI.Logger().Error(fmt.Sprintf("failed to notify admin of system_packages install failure for %q: %v", pkg, err))
	}
}

// notifyPluginPaymentRequired raises an admin notification that a store plugin was
// disabled because its purchase has lapsed (it is kept on disk but not loaded).
// Always notifies (regardless of env) since it is an enforcement action the
// operator must see; the cause is also logged. Admin-only — never surfaced on the
// captive portal. Names only the package, per the error-message hygiene rules.
func notifyPluginPaymentRequired(g *api.CoreGlobals, pkg string) {
	if err := g.CoreAPI.Logger().Error(fmt.Sprintf("store plugin %q disabled: purchase required", pkg)); err != nil {
	}

	subject := g.CoreAPI.Translate("error", "Plugin disabled — purchase required")
	content := g.CoreAPI.Translate("error", "The plugin <% .pkg %> was disabled because it is not purchased for this machine. Purchase it to re-enable it on the next restart", "pkg", pkg)

	if err := g.CoreAPI.Notification().AddNotification(context.Background(), sdkapi.AddNotificationParams{
		Subject: subject,
		Content: content,
		Type:    sdkapi.NotificationTypeError,
	}); err != nil {
		g.CoreAPI.Logger().Error(fmt.Sprintf("failed to notify admin that store plugin %q requires payment: %v", pkg, err))
	}
}

// notifyPluginUnavailable raises an admin notification that a store plugin was
// disabled because the cloud reports it as no longer available — its developer has
// withdrawn it from the store. Distinct from notifyPluginPaymentRequired: there is
// nothing the operator can purchase to bring it back, so the wording must not imply
// payment. Always notifies (regardless of env) since it is an enforcement action the
// operator must see; the cause is also logged. Admin-only — never surfaced on the
// captive portal. Names only the package, per the error-message hygiene rules.
func notifyPluginUnavailable(g *api.CoreGlobals, pkg string) {
	g.CoreAPI.Logger().Error(fmt.Sprintf("store plugin %q disabled: no longer available in the store", pkg))

	subject := g.CoreAPI.Translate("error", "Plugin disabled — no longer available")
	content := g.CoreAPI.Translate("error", "The plugin <% .pkg %> was disabled because it has been withdrawn from the store by its developer and can no longer be installed or updated", "pkg", pkg)

	if err := g.CoreAPI.Notification().AddNotification(context.Background(), sdkapi.AddNotificationParams{
		Subject: subject,
		Content: content,
		Type:    sdkapi.NotificationTypeError,
	}); err != nil {
		g.CoreAPI.Logger().Error(fmt.Sprintf("failed to notify admin that store plugin %q is no longer available: %v", pkg, err))
	}
}

// notifyOffline raises an admin notification that the machine has no internet, so
// the operator knows network setup may be required. Admin-only — never surfaced on
// the captive portal. Fires on every internet-down transition (netmon collapses
// flaps into a single event), including the first probe at boot if it boots offline.
func notifyOffline(g *api.CoreGlobals) {
	if err := g.CoreAPI.Logger().Info("online monitor: machine is offline"); err != nil {
	}

	subject := g.CoreAPI.Translate("warning", "No internet connection")
	content := g.CoreAPI.Translate("warning", "The machine has no internet connection. Some features that depend on the cloud are unavailable until connectivity is restored")

	if err := g.CoreAPI.Notification().AddNotification(context.Background(), sdkapi.AddNotificationParams{
		Subject: subject,
		Content: content,
		Type:    sdkapi.NotificationTypeWarn,
	}); err != nil {
		g.CoreAPI.Logger().Error(fmt.Sprintf("failed to notify admin that the machine is offline: %v", err))
	}
}
