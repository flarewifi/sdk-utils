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
		if err := InitPlugins(g); err != nil {
			// In development a plugin that fails to compile or load aborts the
			// boot: we never signal bootCh, so the app routes never come up, and
			// the process exits so the failure is visible (docker logs / reflex)
			// instead of booting with a broken plugin set. Production returns nil
			// from the load loop (it notifies + recovers from backup and keeps
			// going), so this exit path is effectively development-only.
			msg := fmt.Sprintf("Boot aborted: plugin initialization failed: %v", err)
			if logErr := g.CoreAPI.Logger().Error(msg); logErr != nil {
			}
			if env.GO_ENV != env.ENV_PRODUCTION {
				fmt.Println(msg)
				os.Exit(1)
			}
		}
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
