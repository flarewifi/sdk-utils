package pkg

import (
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func GetInfoFromDef(def sdkutils.PluginSrcDef) (info sdkutils.PluginInfo, err error) {
	path, ok := FindDefInstallPath(def)
	if !ok {
		return info, ErrNotInstalled
	}

	return sdkutils.GetPluginInfoFromPath(path)
}

func GetCoreInfo() sdkutils.PluginInfo {
	pluginJsonPath := filepath.Join(sdkutils.PathCoreDir, "plugin.json")
	var pluginDef sdkutils.PluginInfo
	if err := sdkutils.JsonRead(pluginJsonPath, &pluginDef); err != nil {
		panic(err)
	}
	return pluginDef
}
