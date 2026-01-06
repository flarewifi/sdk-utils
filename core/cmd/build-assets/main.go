package main

import (
	"core/utils/plugins"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func main() {
	pluginDirs := []string{sdkutils.PathCoreDir}

	// Get unfiltered list of plugins (local + system)
	localPlugins := plugins.LocalPluginSrcDefs()
	systemPlugins := plugins.SystemPluginSrcDefs()

	for _, def := range append(systemPlugins, localPlugins...) {
		pluginDir, err := filepath.Abs(def.LocalPath)
		if err != nil {
			panic(err)
		}
		pluginDirs = append(pluginDirs, pluginDir)
	}

	for _, pluginDir := range pluginDirs {
		if err := plugins.BuildAssets(pluginDir); err != nil {
			panic(err)
		}
	}
}
