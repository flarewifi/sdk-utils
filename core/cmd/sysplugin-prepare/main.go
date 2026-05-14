package main

import (
	tools "core/utils"
	"core/utils/plugins"
	"fmt"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// sysplugin-prepare prepares the build host for a non-mono build with
// statically-linked system plugins. For every plugin under plugins/system/ it:
//
//  1. Enforces the three-file plugin entry-point contract:
//       - Adds //go:build !mono to main.go if no build tag is present.
//       - Generates main_mono.go from main.go (if absent).
//       - Generates system/main.go from main.go (if absent).
//     Once generated, main_mono.go and system/main.go are author-owned —
//     subsequent runs leave them alone. Delete to force regeneration.
//  2. Validates that all three required files exist after the generation
//     pass; panics with a clear message otherwise.
//  3. Runs BuildQueries / BuildTemplates / BuildAssets so the static-linked
//     code sees fully-generated sqlc / templ / asset manifests at compile time.
//  4. Writes the plugin's metadata entry and copies its runtime data tree into
//     sdkutils.PathPluginInstallDir/<pkg> (resources, migrations, plugin.json,
//     etc.) — the LoadSystemPlugins loader still reads its pluginDir from disk
//     even though the Go code is statically linked.
//
// Devel/local/git/store plugins are NOT processed here. They continue to be
// compiled as plugin.so on the remote plugin-build worker and copied into
// plugins/installed/ by the software-release builder's handlePlugins step.
//
// Intentionally does NOT call config.ResetPluginsConfig — that would wipe the
// metadata entries written by the rest of the non-mono release flow for the
// other plugin source types.
func main() {
	fmt.Println("Preparing system plugins (enforce three-file contract, build assets/sqlc/templ, install data)...")

	tools.CreateGoWorkspace()
	tools.CreateSystemPluginInit()

	systemPlugins := plugins.SystemPluginSrcDefs()

	for _, p := range systemPlugins {
		pluginDir := p.LocalPath

		info, err := sdkutils.GetPluginInfoFromPath(pluginDir)
		if err != nil {
			panic(fmt.Errorf("failed to get plugin info from %s: %w", pluginDir, err))
		}

		// Three-file contract enforcement. Order matters: the build tag must
		// be on main.go before snapshotting so the snapshots are taken from
		// a tagged source. (Strictly the snapshots strip the tag anyway, but
		// the ordering keeps the contract self-consistent at all times.)
		tools.EnsureMainGoBuildTag(pluginDir)
		tools.EnsurePluginMainMono(pluginDir)
		tools.EnsurePluginSystemFiles(pluginDir)
		tools.ValidatePluginContract(pluginDir)

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
			panic(fmt.Errorf("failed to write metadata for system plugin %s: %w", info.Name, err))
		}

		installPath := filepath.Join(sdkutils.PathPluginInstallDir, info.Package)
		fmt.Printf("Copying system plugin %s to installed plugins directory: %s\n", info.Name, installPath)

		if err := sdkutils.CopyPluginFilesMono(pluginDir, installPath); err != nil {
			panic(err)
		}
	}

	fmt.Println("System plugin preparation completed successfully.")
}
