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
	fmt.Println("Building the monolithic binary...")

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
			panic(err)
		}

		installPath := filepath.Join(sdkutils.PathPluginInstallDir, info.Package)
		fmt.Printf("Copying plugin %s to installed plugins directory: %s\n", info.Name, installPath)

		if err := sdkutils.CopyPluginFilesMono(pluginDir, installPath); err != nil {
			panic(err)
		}

		pluginDirs = append(pluginDirs, pluginDir)
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

	fmt.Println("Building flare CLI for mono with:")
	sdkutils.PrettyPrint(opts)

	if err := sdkutils.BuildGoModule(flareCliMain, "bin/flare", opts); err != nil {
		panic(fmt.Errorf("failed to build flare CLI: %w", err))
	}

	tmpOutputDir := filepath.Join(sdkutils.PathTmpDir, ".tmp-mono-output")
	defer os.RemoveAll(tmpOutputDir)

	files := []string{
		"bin/flare",
		"core/go.mod",
		"core/plugin.json",
		"core/sqlc.yml",
		"core/resources/assets/dist",
		"core/resources/assets/public",
		"core/resources/migrations",
		"core/resources/translations",
		"defaults",
		"data/config",
		"plugins/installed",
		"scripts",
		"start.sh",
	}

	for _, f := range files {
		// Copy app files to tmp output directory
		if err := sdkutils.FsCopy(filepath.Join(sdkutils.PathAppDir, f), filepath.Join(tmpOutputDir, f)); err != nil {
			panic(fmt.Errorf("failed to copy %s to tmp output directory: %w", f, err))
		}
	}

	// Copy data/config to ./defaults
	// if err := sdkutils.FsCopy(filepath.Join(sdkutils.PathAppDir, "data/config"), filepath.Join(tmpOutputDir, "defaults")); err != nil {
	// 	panic(fmt.Errorf("failed to copy config to defaults: %w", err))
	// }

	// Create database config
	// dbcfg := `{"sqlite_path": "data/db/database.sqlite"}`
	// if err := sdkutils.FsWriteFile(filepath.Join(tmpOutputDir, "defaults/database.json"), []byte(dbcfg)); err != nil {
	// 	panic(fmt.Errorf("failed to write database config: %w", err))
	// }

	// Remove ./data directory
	// if err := os.RemoveAll(filepath.Join(tmpOutputDir, "data")); err != nil {
	// 	panic(fmt.Errorf("failed to remove data directory: %w", err))
	// }

	outputDir := filepath.Join(sdkutils.PathAppDir, "output/mono-bin-files")
	fmt.Println("Moving mono binary files to output directory: " + outputDir)

	output := &sdkutils.BuildOutput{
		SourceDir: tmpOutputDir,
		OutputDir: outputDir,
		Files:     files,
	}

	if err := output.Run(); err != nil {
		panic(fmt.Errorf("failed to copy mono binary files to output directory: %w", err))
	}

	fmt.Println("Mono files creation completed successfully.")
}
