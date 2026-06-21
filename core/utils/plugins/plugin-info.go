package plugins

import (
	"core/utils/config"
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func GetInfoFromDef(def sdkutils.PluginSrcDef) (info sdkutils.PluginInfo, err error) {
	path, ok := FindDefInstallPath(def)
	if !ok {
		return info, ErrNotInstalled
	}

	return sdkutils.GetPluginInfoFromPath(path)
}

func GetPluginDef(pkg string) (def sdkutils.PluginSrcDef, err error) {
	pluginsCfg, err := config.ReadPluginsConfig()
	if err != nil {
		return def, err
	}

	for _, metadata := range pluginsCfg.Metadata {
		if metadata.Package == pkg {
			return metadata.Def, nil
		}
	}

	return def, ErrNotInstalled
}

func GetCoreInfo() sdkutils.PluginInfo {
	pluginJsonPath := filepath.Join(sdkutils.PathCoreDir, "plugin.json")
	var pluginDef sdkutils.PluginInfo
	if err := sdkutils.JsonRead(pluginJsonPath, &pluginDef); err != nil {
		panic(err)
	}
	return pluginDef
}
