package main

import (
	"fmt"
	"path/filepath"
	"tools"
	"tools/config"
	"tools/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func main() {
	fmt.Println("Preparing monolithic build files (assets/sqlc/templ)...")

	tools.CreateGoWorkspace()
	tools.CreateMonoPluginInit()

	localPlugins := plugins.LocalPluginSrcDefs()
	systemPlugins := plugins.SystemPluginSrcDefs()
	allPlugins := append(localPlugins, systemPlugins...)

	// Reset data/config/plugins.json
	if err := config.ResetPluginsConfig(); err != nil {
		panic(err)
	}

	// Build plugin assets and move to plugins/installed
	for _, p := range allPlugins {
		pluginDir := p.LocalPath

		info, err := sdkutils.GetPluginInfoFromPath(pluginDir)
		if err != nil {
			panic(fmt.Errorf("failed to get plugin info from %s: %w", pluginDir, err))
		}

		if err := plugins.BuildQueries(pluginDir); err != nil {
			panic(err)
		}

		if err := plugins.BuildTemplates(pluginDir); err != nil {
			panic(err)
		}

		if err := plugins.BuildAssets(pluginDir); err != nil {
			panic(err)
		}

		if err := plugins.WriteMetadata(p, info.Package); err != nil {
			panic(fmt.Errorf("failed to write metadata for plugin %s: %w", info.Name, err))
		}

		installPath := filepath.Join(sdkutils.PathPluginInstallDir, info.Package)
		fmt.Printf("Copying plugin %s to installed plugins directory: %s\n", info.Name, installPath)

		if err := sdkutils.CopyPluginFilesMono(pluginDir, installPath); err != nil {
			panic(err)
		}
	}

	// Build core assets
	if err := plugins.BuildAssets(sdkutils.PathCoreDir); err != nil {
		panic(err)
	}
	if err := plugins.BuildQueries(sdkutils.PathCoreDir); err != nil {
		panic(err)
	}
	if err := plugins.BuildTemplates(sdkutils.PathCoreDir); err != nil {
		panic(err)
	}

	fmt.Println("Mono build preparation completed successfully.")
}
