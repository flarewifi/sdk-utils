package boot

import (
	"context"
	"os"
	"time"

	"core/internal/api"
	"core/internal/web"
	"core/internal/web/router"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func InitHttpServer(g *api.CoreGlobals, bootCh chan struct{}) {
	web.SetupBootRoutes(g)
	server := web.StartServer(router.BootingRouter, false)

	// Serve the booting page over HTTPS too, not just HTTP, so clients that reach
	// the machine over TLS during boot (e.g. https:// captive-portal probes, or a
	// browser that cached the HTTPS portal origin) get the booting page instead of
	// a refused connection. Best-effort: a cert error here must not block the HTTP
	// booting page, mirroring the post-boot call below which also ignores the error.
	web.StartHTTPSServer(router.BootingRouter)

	// Wait for boot process to complete
	<-bootCh

	// Remove software update indicator files
	os.Remove(sdkutils.PathIsUpdated)
	os.Remove(sdkutils.PathIsReverted)

	// Gracefully shut down the server to clear booting routes
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.Shutdown(ctx)

	// Stop the booting HTTPS listener before restarting it on RootRouter below.
	// StartHTTPSServer is a no-op while the server is already running and the live
	// server keeps the handler it was started with (BootingRouter), so without this
	// stop the app would keep serving booting routes over TLS.
	web.StopHTTPSServer()

	// Both booting listeners are down now, so /boot/progress is gone and nothing
	// reads the boot timeline again. Free it (the per-plugin checklist can hold a
	// couple dozen steps) and drop the shut-down booting server — InitHttpServer
	// never returns (it blocks on StartServer below), so without this both would
	// stay reachable, and thus alive, for the machine's entire uptime.
	g.BootProgress.Release()
	server = nil

	// Restart the server with all routes
	web.SetupAppRoutes(g)

	// Notify that the server is starting
	startedAt := time.Now().Format(time.RFC3339)
	os.WriteFile(sdkutils.PathServerUp, []byte(startedAt), sdkutils.PermFile)

	web.StartHTTPSServer(router.RootRouter)
	web.StartServer(router.RootRouter, true)
}
