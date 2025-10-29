package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"tools"
	"tools/config"
	"tools/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func main() {
	tools.CreateGoWorkspace()
	tools.CreateMonoPluginInit()

	localPlugins := plugins.LocalPluginSrcDefs()
	systemPlugins := plugins.SystemPluginSrcDefs()
	allPlugins := append(localPlugins, systemPlugins...)

	// Reset data/config/plugins.json
	if err := config.ResetPluginsConfig(); err != nil {
		panic(err)
	}

	pluginDirs := []string{sdkutils.PathCoreDir}

	if err := config.ResetPluginsConfig(); err != nil {
		panic(fmt.Errorf("failed to reset plugins config: %w", err))
	}

	// Build plugin assets and move to plugins/installed
	for _, p := range allPlugins {
		pluginDir := p.LocalPath

		info, err := sdkutils.GetPluginInfoFromPath(pluginDir)
		if err != nil {
			panic(err)
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
			panic(fmt.Errorf("failed to write plugin metadata for %s: %w", info.Name, err))
		}

		installPath := filepath.Join(sdkutils.PathPluginInstallDir, info.Package)
		fmt.Printf("Copying plugin %s to installed plugins directory: %s\n", info.Name, installPath)

		if err := sdkutils.CopyPluginFilesMono(pluginDir, installPath); err != nil {
			panic(err)
		}

		pluginDirs = append(pluginDirs, pluginDir)
	}

	if err := plugins.BuildGlobalAssets(pluginDirs); err != nil {
		panic(err)
	}

	if err := plugins.BuildAssets(sdkutils.PathCoreDir); err != nil {
		panic(err)
	}

	goArch := os.Getenv("GO_ARCH")
	if goArch == "" {
		goArch = runtime.GOARCH
	}

	flareCliMain := filepath.Join(sdkutils.PathCoreDir, "internal/cli/main.go")
	opts := sdkutils.GoBuildOpts{
		BuildTags: os.Getenv("GO_TAGS") + " mono sqlite",
		GoArch:    goArch,
	}

	if err := sdkutils.BuildGoModule(flareCliMain, "bin/flare", opts); err != nil {
		panic(err)
	}

	outputDir := filepath.Join(sdkutils.PathAppDir, "output/mono-bin-files")
	fmt.Println("Moving mono binary files to output directory: " + outputDir)

	files := []string{
		"bin/flare",
		"core/go.mod",
		"core/plugin.json",
		"core/sqlc.yml",
		"core/resources/assets/dist",
		"core/resources/assets/public",
		"core/resources/migrations",
		"core/resources/translations",
		"data/config/plugins.json",
		"defaults",
		"plugins/installed",
		"scripts",
		"start.sh",
	}

	for _, f := range files {
		fmt.Printf("Copying '%s' to '%s'\n", f, sdkutils.StripRootPath(outputDir))
		if err := sdkutils.FsCopy(filepath.Join(sdkutils.PathAppDir, f), filepath.Join(outputDir, f)); err != nil {
			panic(err)
		}
	}

	// Create database config
	dbcfg := `{"sqlite_path": "data/db/database.sqlite"}`
	if err := os.WriteFile(filepath.Join(outputDir, "data/config/database.json"), []byte(dbcfg), 0644); err != nil {
		panic(err)
	}

	fmt.Println("Mono files creation completed successfully.")
}
