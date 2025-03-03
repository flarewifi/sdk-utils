package tools

import (
	"core/internal/utils/plugins"
	"fmt"
	"log"
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

	// insert libs paths
	libs := []string{}
	if err := sdkutils.FsListDirs("sdk/libs", &libs, false); err != nil {
		log.Println(err)
	}

	for _, lib := range libs {
		goWork += "\n    ./" + lib
	}

	// insert plugin paths
	pluginSearchPaths := []string{"plugins/system", "plugins/local"}
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
}
