package boot

import (
	"core/internal/api"
	"log"
)

func Init(g *api.CoreGlobals) {
	bootCh := make(chan struct{})

	InitDirs()

	go func() {
		log.Println("Initializing database...")
		g.Db.WaitReady()

		log.Println("Database is ready.")
		InitOpkg()
		RunCoreMigrations(g)
		InitPlugins(g)
		InitAccounts()
		if err := InitNetwork(); err != nil {
			log.Println("Error initializing network:", err)
		}

		bootCh <- struct{}{}
	}()

	InitHttpServer(g, bootCh)
}
