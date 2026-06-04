package boot

import (
	"context"
	"core/internal/api"
	"core/internal/jobs"
	"core/internal/modules/activation"
	coretheme "core/internal/theme"
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
		InitPlugins(g)
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
