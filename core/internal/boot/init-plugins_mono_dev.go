//go:build dev && mono

package boot

// import (
// 	"core/internal/api"
// 	"core/internal/utils/plugins"
// 	"fmt"

// 	sdkutils "github.com/flarehotspot/sdk-utils"
// )

// func InitPlugins(g *api.CoreGlobals) {
// 	systemDefs := plugins.SystemPluginSrcDefs()
// 	localDefs := plugins.LocalPluginSrcDefs()
// 	for _, def := range append(systemDefs, localDefs...) {
// 		info, err := sdkutils.GetPluginInfoFromPath(def.LocalPath)
// 		if err != nil {
// 			fmt.Printf("Error getting plugin info from path: %s\n", err)
// 			continue
// 		}
// 		dir := def.LocalPath

// 		p := api.NewPluginApi(dir, info, g.PluginMgr, g.TrafficMgr)
// 		g.PluginMgr.RegisterPlugin(p)
// 	}
// }
