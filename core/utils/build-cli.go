package tools

import (
	"core/utils/plugins"
	"core/utils/tags"
	"fmt"
	"os"

	sdkutils "github.com/flarewifi/sdk-utils"
)

type FlareCliBuild struct {
	GOOS   string
	GOARCH string
	File   string
}

func BuildFlareCLI() {
	fmt.Println("Building flare CLI...")

	cliFile := "core/internal/cli/main.go"
	cliPath := "bin/flare"
	workdir, _ := os.Getwd()
	opts := sdkutils.GoBuildOpts{
		GoBinPath: plugins.GoBin(),
		WorkDir:   workdir,
		BuildTags: tags.GetBuildTags(),
	}

	if err := sdkutils.BuildGoModule(cliFile, cliPath, opts); err != nil {
		panic(err)
	}

	fmt.Printf("Flare CLI built at: %s\n", cliPath)
}
