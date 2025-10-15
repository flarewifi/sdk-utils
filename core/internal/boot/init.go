package boot

import (
	"core/internal/api"
	"core/internal/utils/plugins"
	"log"
	"time"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func Init(g *api.CoreGlobals) {
	bootCh := make(chan struct{})

	InitDirs()

	go func() {
		log.Println("Initializing database...")
		g.Db.WaitReady()
		log.Println("Database is ready.")

		plugins.LinkNodeModulesLib(sdkutils.PathAppDir)
		InitOpkg()
		RunCoreMigrations(g)
		InitPlugins(g)
		InitAccounts()
		if err := InitNetwork(); err != nil {
			log.Println("Error initializing network:", err)
		}

		time.Sleep(8 * time.Second) // Simulate some boot delay

		bootCh <- struct{}{}
	}()

	InitHttpServer(g, bootCh)
}
