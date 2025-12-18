package main

import (
	"fmt"
	"path/filepath"
	"core/tools/tags"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func main() {
	fmt.Println("Building the monolithic binary...")

	goTags := tags.GetBuildTags()
	env := []string{
		"GO_TAGS=" + goTags,
	}

	flareCliMain := filepath.Join(sdkutils.PathCoreDir, "internal/cli/main.go")
	opts := sdkutils.GoBuildOpts{
		BuildTags: goTags,
		Env:       env,
	}

	fmt.Println("Building flare CLI for mono with:")
	sdkutils.PrettyPrint(opts)

	if err := sdkutils.BuildGoModule(flareCliMain, "bin/flare", opts); err != nil {
		panic(fmt.Errorf("failed to build flare CLI: %w", err))
	}

	fmt.Println("Mono binary build completed successfully.")
}
