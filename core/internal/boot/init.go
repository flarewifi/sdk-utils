package boot

import (
	"fmt"
	"time"

	"core/internal/api"
	"core/internal/utils/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func Init(g *api.CoreGlobals) {
	bp := g.BootProgress
	now := time.Now()

	InitDirs()

	bootServer := InitBootRouteHttpServer(g)

	go func() {
		plugins.LinkNodeModulesLib(sdkutils.PathAppDir)

		InitOpkg(bp)

		bp.AppendLog("Running core migrations...")
		time.Sleep(30 * time.Second)
		RunCoreMigrations(g)

		bp.AppendLog("Initializing plugins...")

		time.Sleep(30 * time.Second)
		InitPlugins(g)

		bp.AppendLog("Initializing admin accounts...")
		time.Sleep(30 * time.Second)
		InitAccounts()

		bp.AppendLog("Setting up network interfaces...")
		time.Sleep(30 * time.Second)
		InitNetwork()

		s := fmt.Sprintf("Done booting in %v", time.Since(now))
		time.Sleep(30 * time.Second)
		bp.AppendLog(s)

		time.Sleep(90 * time.Second)
		bp.Done(nil)
	}()

	InitAllRoutesHttpServer(g, bootServer)
}
