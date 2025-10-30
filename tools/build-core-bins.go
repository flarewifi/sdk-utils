package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"tools/env"
	"tools/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func BuildCoreBins() {
	BuildFlareCLI()
	BuildCore()

	goversion := sdkutils.GO_VERSION
	tags := sdkutils.Slugify(env.BuildTags, "-")
	info := plugins.GetCoreInfo()

	build := &BuildOutput{
		OutputDirName: filepath.Join("core-binaries", fmt.Sprintf("core_arch_bin-%s-%s-go%s-%s", info.Version, sdkutils.GOARCH, goversion, tags)),
		Files: []string{
			"bin/flare",
			"core/plugin.so",
			"core/resources/assets/dist",
			"core/resources/assets/public",
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
