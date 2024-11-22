package main

import (
	"core/build/tools"
	"core/internal/utils/pkg"
	"os"
	"path/filepath"

	sdkfs "github.com/flarehotspot/go-utils/fs"
	sdkpaths "github.com/flarehotspot/go-utils/paths"
)

func main() {
	tools.SyncCoreVersion()
	tools.SyncGoVersion()
	version := pkg.CoreInfo().Version
	releaseNotePath := filepath.Join(sdkpaths.CoreDir, "build", "release-notes", version+".md")
	if !sdkfs.Exists(releaseNotePath) {
		if err := os.WriteFile(releaseNotePath, []byte("## "+version+"\n\n"), 0644); err != nil {
			panic(err)
		}
		return
	}
}
