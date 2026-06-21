package main

import (
	"flag"
	"path/filepath"

	"core/utils/plugins"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func main() {
	// --core-only builds just the core's own assets (resources/assets/dist),
	// skipping local/system plugins. The dev hot-reload loop (start-dev.sh) uses
	// this because `flare build-plugins` already rebuilds plugin assets — without
	// the flag every save would rebuild all plugin assets twice. Full builds
	// (mono/openwrt dev scripts, release builds) call this without the flag.
	coreOnly := flag.Bool("core-only", false, "Build only the core assets, skipping local/system plugins")
	flag.Parse()

	pluginDirs := []string{sdkutils.PathCoreDir}

	if !*coreOnly {
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
	}

	for _, pluginDir := range pluginDirs {
		if err := plugins.BuildAssets(pluginDir); err != nil {
			panic(err)
		}
	}
}
