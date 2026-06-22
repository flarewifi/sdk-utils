package boot

import (
	"context"
	"fmt"
	"os"
	"time"

	"core/internal/api"
	"core/internal/jobs"
	"core/internal/modules/activation"
	"core/internal/modules/netmon"
	coretheme "core/internal/theme"
	"core/utils/env"
)

// How long the boot sequence waits for internet before falling back to
// offline-first boot, and how often it re-probes while waiting. Only reached when
// a plugin still has unprovisioned system_packages/install scripts (first boot or
// after an upgrade); an already-provisioned machine never waits. The first probe
// is immediate, so an already-online machine proceeds at once.
const (
	provisionOnlineWait    = 60 * time.Second
	provisionProbeInterval = 3 * time.Second
)

func Init(g *api.CoreGlobals) {
	bootCh := make(chan struct{})

	InitDirs()

	go func() {
		ctx := context.Background()

		g.BootProgress.Advance(g.CoreAPI.Translate("info", "Preparing database"))
		g.Database.WaitReady()

		InitOpkg()
		RunCoreMigrations(g)
		InitTranslations()
		coretheme.SetAdminTheme(g.CoreAPI)
		coretheme.SetPortalTheme(g.CoreAPI)

		g.BootProgress.Advance(g.CoreAPI.Translate("info", "Loading plugins"))
		// InitPlugins now only LOADS plugins (maps each .so, resolves Init); it
		// does not run Init. A load failure (e.g. a stale/ABI-broken .so) still
		// aborts the boot in development so the failure is visible in docker logs /
		// reflex instead of booting a broken plugin set; production notifies +
		// recovers from backup and keeps going.
		if err := InitPlugins(g); err != nil {
			msg := fmt.Sprintf("Boot aborted: plugin load failed: %v", err)
			if logErr := g.CoreAPI.Logger().Error(msg); logErr != nil {
			}
			if env.GO_ENV != env.ENV_PRODUCTION {
				fmt.Println(msg)
				os.Exit(1)
			}
		}
		// Run Init for plugins that need no internet-dependent provisioning, so the
		// device's core function (e.g. the captive portal) comes up offline. Plugins
		// with unprovisioned system_packages/preinstall are skipped here and have
		// their Init run later, after the online monitor provisions them.
		if err := InitLoadedPlugins(g); err != nil {
			msg := fmt.Sprintf("Boot aborted: plugin initialization failed: %v", err)
			if logErr := g.CoreAPI.Logger().Error(msg); logErr != nil {
			}
			if env.GO_ENV != env.ENV_PRODUCTION {
				fmt.Println(msg)
				os.Exit(1)
			}
		}

		// Register with the cloud BEFORE the network-dependent provisioning below.
		// Activation is lightweight (a few local file reads + one Twirp call) and is
		// what creates the machine's row server-side and makes it appear/manageable in
		// the fleet. Provisioning installs each plugin's system_packages via opkg/pip
		// and can stall for minutes — or OOM the process — on constrained mono devices.
		// Sequenced after provisioning (as it used to be), such a machine would die
		// mid-install and NEVER register, silently missing from the machines list.
		// Kicking it here, in its own guarded goroutine, lets registration complete in
		// parallel and survive a later provisioning failure; the online monitor
		// re-attempts it on every reconnect (see StartOnlineMonitor).
		activation.CheckActivationFileExists()
		StartActivation()

		// Network-dependent work (each plugin's system_packages, its preinstall/
		// postinstall scripts, and the deferred Init of plugins that need them)
		// cannot run during the offline part of boot — opkg/pip need the feed/PyPI.
		//
		// When a plugin still has such work pending (first boot or after an upgrade),
		// hold the booting page through it: wait a bounded time for internet, then run
		// the provisioning pass synchronously so the page shows "waiting for internet"
		// and "installing packages (N/M)" instead of dropping the user into a
		// half-initialized app. If the machine stays offline past the wait, fall back
		// to offline-first boot — the online monitor (started just below) runs the
		// pass the moment connectivity appears and retries on reconnect.
		//
		// Doing the first pass here, before StartOnlineMonitor, also avoids racing the
		// monitor's immediate internet-up handler; its later pass is an idempotent
		// no-op because the version-pinned markers are already written.
		//
		// RunBootProvisioning bounds the wait (provisionBootCap): provisioning runs
		// opkg/pip and a deferred plugin's Init, any of which can stall for minutes on
		// a flaky link — and boot must never hang on it, or the captive portal never
		// comes up. Past the cap, boot proceeds offline-first and the pass finishes in
		// the background under the provisioning guard.
		if NeedsProvisionAny(g) {
			g.BootProgress.Advance(g.CoreAPI.Translate("info", "Waiting for internet connection"))
			if netmon.WaitOnline(ctx, provisionOnlineWait, provisionProbeInterval) {
				g.BootProgress.Advance(g.CoreAPI.Translate("info", "Installing required packages"))
				RunBootProvisioning(g)
			}
		}

		StartOnlineMonitor(g)

		g.BootProgress.Advance(g.CoreAPI.Translate("info", "Finalizing startup"))
		InitAssets(g)
		InitAccounts()
		if err := InitNetwork(); err != nil {
		} else {
			api.RunNetworkReadyCallbacks(g.CoreAPI.Logger())
		}

		// Initialize sessions manager
		if err := g.ClientMgr.Init(ctx); err != nil {
		} else {
		}

		// Start jobs
		jobs.Init(g)

		g.BootProgress.Done()
		bootCh <- struct{}{}
	}()

	InitHttpServer(g, bootCh)
}
