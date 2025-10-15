package boot

import (
	"context"
	"log"
	"os"
	"time"

	"core/internal/api"
	webutil "core/internal/utils/web"
	"core/internal/web"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func InitHttpServer(g *api.CoreGlobals, bootCh chan struct{}) {
	web.SetupBootRoutes(g)
	server := web.StartServer(webutil.BootingRouter, false)

	// Wait for boot process to complete
	<-bootCh

	// Remove software update indicator files
	os.Remove(sdkutils.PathIsUpdated)
	os.Remove(sdkutils.PathIsReverted)

	log.Println("Boot progress done. Need to restart server...")

	// Gracefully shut down the server to clear booting routes
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		log.Printf("Server shutdown error: %v\n", err)
	} else {
		log.Println("Server gracefully stopped")
	}

	// Restart the server with all routes
	web.SetupAppRoutes(g)
	log.Println("Starting server...")
	web.StartServer(webutil.RootRouter, true)
}
