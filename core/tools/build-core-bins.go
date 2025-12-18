package tools

import (
	"os"
	"path/filepath"
	"core/tools/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func BuildCoreBins() {
	BuildFlareCLI()
	BuildCore()

	build := &sdkutils.BuildOutput{
		OutputDir: filepath.Join(sdkutils.PathAppDir, "output/core-binaries"),
		Files: []string{
			"bin/flare",
			"core/plugin.so",
			"core/resources",
		},
	}

	if err := build.Run(); err != nil {
		panic(err)
	}
}

func BuildCore() {

	if err := plugins.BuildTemplates(sdkutils.PathCoreDir); err != nil {
		panic(err)
	}

	if err := plugins.BuildQueries(sdkutils.PathCoreDir); err != nil {
		panic(err)
	}

	if err := plugins.BuildAssets(sdkutils.PathCoreDir); err != nil {
		panic(err)
	}

	workdir := filepath.Join(sdkutils.PathTmpDir, "b/core", sdkutils.RandomStr(16))
	defer os.RemoveAll(workdir)
	if err := plugins.BuildPluginSo(sdkutils.PathCoreDir, workdir); err != nil {
		panic(err)
	}
}
