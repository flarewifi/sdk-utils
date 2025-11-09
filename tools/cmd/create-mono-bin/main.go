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

	env := []string{
		"GOARCH=" + goArch,
	}

	if goArch == "amd64" {
		env = append(env, "CGO_ENABLED=0")
	}

	flareCliMain := filepath.Join(sdkutils.PathCoreDir, "internal/cli/main.go")
	opts := sdkutils.GoBuildOpts{
		BuildTags: os.Getenv("GO_TAGS") + " mono sqlite",
		Env:       env,
		// GoArch:    goArch,
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

	outputDir := filepath.Join(sdkutils.PathAppDir, "output/mono-bin-files")
	for _, f := range files {
		// Copy app files to tmp output directory
		if err := sdkutils.FsCopy(filepath.Join(sdkutils.PathAppDir, f), filepath.Join(outputDir, f)); err != nil {
			panic(fmt.Errorf("failed to copy %s to tmp output directory: %w", f, err))
		}
	}

	fmt.Println("Mono files creation completed successfully.")
}
