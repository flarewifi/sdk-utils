package boot

import (
	"context"
	"fmt"
	"os"

	"core/internal/api"
	"core/internal/jobs"
	"core/internal/modules/activation"
	coretheme "core/internal/theme"
	"core/utils/env"
)

func Init(g *api.CoreGlobals) {
	bootCh := make(chan struct{})

	InitDirs()

	go func() {
		g.Database.WaitReady()

		InitOpkg()
		RunCoreMigrations(g)
		InitTranslations()
		coretheme.SetAdminTheme(g.CoreAPI)
		coretheme.SetPortalTheme(g.CoreAPI)
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
		// Network-dependent work (each plugin's system_packages, its preinstall/
		// postinstall scripts, and the deferred Init of plugins that need them)
		// cannot run during the offline part of boot — opkg/pip need the feed/PyPI.
		// The online monitor runs it the moment the device has internet (immediately
		// if it boots already-online) and retries on reconnect.
		StartOnlineMonitor(g)
		InitAssets(g)
		InitAccounts()
		if err := InitNetwork(); err != nil {
		} else {
			api.RunNetworkReadyCallbacks(g.CoreAPI.Logger())
		}

		// Initialize sessions manager
		ctx := context.Background()
		if err := g.ClientMgr.Init(ctx); err != nil {
		} else {
		}

		// Initialize activation after everything else is ready
		// First perform optimistic check (synchronous) for immediate activation state
		activation.CheckActivationFileExists()
		// Then run full validation in background
		go activation.Validate()

		// Start jobs
		jobs.Init(g)

		bootCh <- struct{}{}
	}()

	InitHttpServer(g, bootCh)
}
