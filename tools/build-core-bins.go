package tools

import (
	"os"
	"path/filepath"
	"tools/plugins"

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
			"core/go.mod",
			"core/go.sum",
			"core/sqlc.yml",
			"core/package.json",
			"core/package-lock.json",
			"core/plugin.json",
			"core/resources",
			"defaults",
			"sdk",
			"scripts",
			"tools/go.mod",
			"tools/go.sum",
			"plugins/system",
			"go.work.default",
			"start.sh",
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

	workdir := filepath.Join(sdkutils.PathTmpDir, "b/core", sdkutils.RandomStr(16))
	defer os.RemoveAll(workdir)
	if err := plugins.BuildPluginSo(sdkutils.PathCoreDir, workdir); err != nil {
		panic(err)
	}
}
