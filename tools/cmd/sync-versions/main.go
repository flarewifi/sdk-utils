package main

import (
	"os"
	"path/filepath"
	"tools"
	"tools/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func main() {
	tools.SyncCoreVersion()
	tools.SyncGoVersion()
	version := plugins.GetCoreInfo().Version
	releaseNotePath := filepath.Join(sdkutils.PathCoreDir, "build", "release-notes", version+".md")

	if !sdkutils.FsExists(releaseNotePath) {
		if err := sdkutils.FsEmptyDir(filepath.Dir(releaseNotePath)); err != nil {
			panic(err)
		}

		if err := os.WriteFile(releaseNotePath, []byte("## "+version+"\n\n"), 0644); err != nil {
			panic(err)
		}
	}
}
