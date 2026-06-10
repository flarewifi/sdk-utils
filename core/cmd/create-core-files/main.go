package main

import (
	tools "core/utils"
	"core/utils/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func main() {
	if err := plugins.BuildAssets(sdkutils.PathCoreDir); err != nil {
		panic(err)
	}

	// Share the core layer + renamed-file set with build-core-bins (the update
	// tarball) so the full image and a core update never drift. Ships the
	// build-appropriate start.sh (staged-overlay applier for non-mono).
	build := &sdkutils.BuildOutput{
		OutputDir: "output/core-files",
		Files:     tools.CoreFileSet(),
		Custom:    tools.CoreCustomFiles(),
	}

	if err := build.Run(); err != nil {
		panic(err)
	}
}
