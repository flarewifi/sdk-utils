package boot

import (
	"time"

	"core/internal/api"
	"core/internal/utils/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func Init(g *api.CoreGlobals) {
	bp := g.BootProgress
	now := time.Now()

	InitDirs()

	go func() {
		bp.AppendLog("Start booting processes...")
		time.Sleep(10 * time.Second)

		bp.AppendLog("Linking node modules...")
		plugins.LinkNodeModulesLib(sdkutils.PathAppDir)
		time.Sleep(10 * time.Second)

		bp.AppendLog("Installing internet packages...")
		InitOpkg(bp)
		time.Sleep(10 * time.Second)

		bp.AppendLog("Running core migrations...")
		RunCoreMigrations(g)
		time.Sleep(10 * time.Second)

		bp.AppendLog("Initializing plugins...")
		InitPlugins(g)
		time.Sleep(10 * time.Second)

		bp.AppendLog("Initializing admin accounts...")
		InitAccounts()
		time.Sleep(10 * time.Second)

		bp.AppendLog("Setting up network interfaces...")
		InitNetwork()
		time.Sleep(10 * time.Second)

		doneMsg := constructDoneMsg(now)
		bp.AppendLog(doneMsg)

		bp.Done(nil)
	}()

	InitHttpServer(g)
}
