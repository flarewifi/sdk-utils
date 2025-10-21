//go:build mono

package boot

import (
	"core/internal/api"
	"core/internal/utils/plugins"
	"fmt"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func InitPlugins(g *api.CoreGlobals) {
	systemDefs := plugins.SystemPluginSrcDefs()
	localDefs := plugins.LocalPluginSrcDefs()
	for _, def := range append(systemDefs, localDefs...) {
		info, err := sdkutils.GetPluginInfoFromPath(def.LocalPath)
		if err != nil {
			fmt.Printf("Error getting plugin info from path: %s\n", err)
			continue
		}
		dir := def.LocalPath

		if err := plugins.BuildAssets(dir); err != nil {
			fmt.Printf("Error building assets for plugin %s: %s\n", info.Name, err)
			continue
		}

		p := api.NewPluginApi(dir, info, g.PluginMgr, g.TrafficMgr)
		g.PluginMgr.RegisterPlugin(p)
	}
}
