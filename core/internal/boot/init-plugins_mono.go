//go:build mono

package boot

import (
	"core/internal/api"
)

func InitPlugins(g *api.CoreGlobals) {
	g.PluginMgr.LoadMonoPlugins(g)
}
