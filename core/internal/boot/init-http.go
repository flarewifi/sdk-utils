package boot

import (
	"context"
	"os"
	"time"

	"core/internal/api"
	"core/internal/web"
	"core/internal/web/router"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func InitHttpServer(g *api.CoreGlobals, bootCh chan struct{}) {
	web.SetupBootRoutes(g)
	server := web.StartServer(router.BootingRouter, false)

	// Wait for boot process to complete
	<-bootCh

	// Remove software update indicator files
	os.Remove(sdkutils.PathIsUpdated)
	os.Remove(sdkutils.PathIsReverted)

	// Gracefully shut down the server to clear booting routes
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.Shutdown(ctx)

	// Restart the server with all routes
	web.SetupAppRoutes(g)

	// Notify that the server is starting
	startedAt := time.Now().Format(time.RFC3339)
	os.WriteFile(sdkutils.PathServerUp, []byte(startedAt), sdkutils.PermFile)

	web.StartHTTPSServer(router.RootRouter)
	web.StartServer(router.RootRouter, true)
}
