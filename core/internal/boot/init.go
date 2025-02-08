package boot

import (
	"fmt"
	"log"
	"time"

	"core/internal/api"
	"core/internal/utils/pkg"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func Init(g *api.CoreGlobals) {
	bp := g.BootProgress
	now := time.Now()

	InitDirs()

	go func() {
		pkg.LinkNodeModulesLib(sdkutils.PathAppDir)

		InitOpkg(bp)

		// delay boot
		// time.Sleep(1000 * 3 * time.Millisecond)

		bp.AppendLog("Running core migrations...")
		RunCoreMigrations(g)

		bp.AppendLog("Initializing plugins...")
		// time.Sleep(1000 * 3 * time.Millisecond)
		InitPlugins(g)

		bp.AppendLog("Initializing admin accounts...")
		InitAccounts()

		// delay boot
		// time.Sleep(1000 * 3 * time.Millisecond)

		bp.AppendLog("Setting up network interfaces...")
		InitNetwork()

		// delay boot
		// time.Sleep(1000 * 3 * time.Millisecond)

		s := fmt.Sprintf("Done booting in %v", time.Since(now))
		bp.AppendLog(s)

		// time.Sleep(1000 * 1 * time.Millisecond)
		bp.Done(nil)

		log.Println("Done booting in", time.Since(now))
	}()

	InitHttpServer(g)
}
