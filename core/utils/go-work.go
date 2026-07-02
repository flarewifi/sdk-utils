package tools

import (
	"core/utils/plugins"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func CreateGoWorkspace() {
	goVersion := sdkutils.GO_SHORT_VERSION

	// Read toolchain directive from go.work.default if it exists
	toolchainLine := ""
	if content, err := os.ReadFile("go.work.default"); err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "toolchain ") {
				toolchainLine = "\n" + line
				break
			}
		}
	}

	if toolchainLine == "" {
		panic("must add toolchain version to go.work.default")
	}

	goWork := fmt.Sprintf(`go %s%s

use (
    ./core
    ./sdk/api
    ./sdk/utils`, goVersion, toolchainLine)

	// Insert plugin paths
	pluginSearchPaths := []string{
		sdkutils.StripRootPath(sdkutils.PathPluginSystemDir),
		sdkutils.StripRootPath(sdkutils.PathPluginLocalDir),
		sdkutils.StripRootPath(sdkutils.PathPluginDevelDir),
	}

	// Track which module path each `use` entry contributes. Two plugin dirs that
	// declare the SAME module (e.g. the devkit theme cloned into
	// data/plugins/system for a build while a developer keeps an iteration copy
	// under data/plugins/devel) would otherwise both be listed, and `go` rejects
	// the workspace with "module <x> appears multiple times". Keep the first and
	// skip the rest, warning so the developer can remove the stray copy. Search
	// order is system → local → devel, so a build's system copy wins; in local dev
	// there should only be one copy anyway.
	seenModule := map[string]string{}

	for _, searchPath := range pluginSearchPaths {
		if sdkutils.FsExists(searchPath) {
			var entries []string
			if err := sdkutils.FsListDirs(searchPath, &entries, false); err != nil {
				continue
			}

			for _, entry := range entries {
				pluginDir, err := sdkutils.FindPluginSrc(entry)
				if err != nil {
					fmt.Printf("%s is not a valid plugin path, skipping...\n", pluginDir)
					continue
				}

				if plugins.ValidateSrcPath(pluginDir) != nil {
					continue
				}

				module, err := goModModulePath(pluginDir)
				if err != nil {
					fmt.Printf("%s has no readable go.mod module, skipping...\n", pluginDir)
					continue
				}
				if first, dup := seenModule[module]; dup {
					fmt.Printf("WARNING: module %q found in both ./%s and ./%s; using the first and skipping the latter. Remove the duplicate copy.\n", module, first, pluginDir)
					continue
				}
				seenModule[module] = pluginDir

				goWork += "\n    ./" + pluginDir
			}
		}
	}

	goWork += "\n)"

	if err := os.WriteFile(filepath.Join("go.work"), []byte(goWork), 0644); err != nil {
		panic(err)
	}

	fmt.Printf("go.work file created:\n%s\n", goWork)

	coreMods, err := plugins.GetRequiredGoModules(filepath.Join(sdkutils.PathAppDir, "core", "go.mod"))
	if err != nil {
		panic(err)
	}

	if err := plugins.UpdateRequiredModules(filepath.Join(sdkutils.PathSdkDir, "api", "go.mod"), coreMods); err != nil {
		panic(err)
	}

	fmt.Println("Updated go.mod file in sdk/api")

	if err := plugins.UpdateRequiredModules(filepath.Join(sdkutils.PathSdkDir, "utils", "go.mod"), coreMods); err != nil {
		panic(err)
	}

	fmt.Println("Updated go.mod file in sdk/utils")
}

// goModModulePath returns the module path declared in pluginDir/go.mod. Used to
// detect two plugin copies that resolve to the same workspace module.
func goModModulePath(pluginDir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(pluginDir, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("no module directive in %s/go.mod", pluginDir)
}
