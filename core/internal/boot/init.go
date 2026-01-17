package boot

import (
	"context"
	"core/internal/api"
	"core/internal/jobs"
	"core/internal/modules/activation"
	"log"
	"os"
	"time"
)

func Init(g *api.CoreGlobals) {
	// Force UTC timezone for the entire application
	// This ensures all time.Now() calls return UTC time
	os.Setenv("TZ", "UTC")
	time.Local = time.UTC
	log.Println("Application timezone set to UTC")

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
