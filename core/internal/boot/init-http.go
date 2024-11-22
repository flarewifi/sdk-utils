package boot

import (
	"context"
	"log"
	"time"

	"core/internal/plugins"
	webutil "core/internal/utils/web"
	"core/internal/web"
)

func InitHttpServer(g *plugins.CoreGlobals) {
	web.SetupBootRoutes(g)
	server := web.StartServer(webutil.BootingRouter, false)

	err := <-g.BootProgress.DONE_C
	if err != nil {
		log.Println("Error while booting:", err)
		// TODO: Show recovery page
		return
	}

	log.Println("Boot progress done. Need to restart server...")

	// Gracefully shut down the server to clear booting routes
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		log.Printf("Server shutdown error: %v\n", err)
	} else {
		log.Println("Server gracefully stopped")
	}

	// Restart the server with all routes
	web.SetupAllRoutes(g)
	log.Println("Starting server...")
	web.StartServer(webutil.RootRouter, true)
}
