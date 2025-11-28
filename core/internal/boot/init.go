package boot

import (
	"context"
	"core/internal/api"
	"core/internal/utils/activation"
	"log"
)

func Init(g *api.CoreGlobals) {
	bootCh := make(chan struct{})

	InitDirs()

	go func() {
		g.Database.WaitReady()
		log.Println("Database is ready.")

		InitOpkg()
		RunCoreMigrations(g)
		InitPlugins(g)
		InitAssets(g)
		InitAccounts()
		if err := InitNetwork(); err != nil {
			log.Println("Error initializing network:", err)
		}

		// Initialize sessions manager
		ctx := context.Background()
		if err := g.ClientMgr.Init(ctx); err != nil {
			log.Println("Error initializing sessions manager:", err)
		} else {
			log.Println("Sessions manager initialized successfully.")
		}

		// Initialize activation after everything else is ready
		go activation.Validate()

		bootCh <- struct{}{}
	}()

	InitHttpServer(g, bootCh)
}
