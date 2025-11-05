//go:build mono

package boot

import (
	"core/internal/api"
	"fmt"
)

func InitPlugins(g *api.CoreGlobals) {
	fmt.Println("Initializing plugins...")
	g.PluginMgr.LoadMonoPlugins(g)
}
