package tools

import (
	"core/internal/utils/plugins"
	"fmt"
	"os"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func CreateGoWorkspace() {
	goVersion := sdkutils.GO_SHORT_VERSION
	goWork := fmt.Sprintf(`go %s

use (
    ./main
    ./core
    ./sdk/api
    ./sdk/utils`, goVersion)

	// Insert plugin paths
	pluginSearchPaths := []string{
		sdkutils.StripRootPath(sdkutils.PathPluginSystemDir),
		sdkutils.StripRootPath(sdkutils.PathPluginLocalDir),
	}

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

				if plugins.ValidateSrcPath(pluginDir) == nil {
					goWork += "\n    ./" + pluginDir
				}
			}
		}
	}

	goWork += "\n)"

	if err := os.WriteFile(filepath.Join("go.work"), []byte(goWork), 0644); err != nil {
		panic(err)
	}

	// fmt.Printf("go.work file created: \n%s\n", goWork)
	fmt.Println("go.work file created.")

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
