package tools

import (
	"core/env"
	"core/internal/utils/pkg"
	"fmt"
	"os"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func BuildCoreBins() {
	BuildFlareCLI()
	BuildCore()

	goversion := sdkutils.GO_VERSION
	tags := sdkutils.Slugify(env.BuildTags, "-")
	info := pkg.GetCoreInfo()

	build := &BuildOutput{
		OutputDirName: filepath.Join("core-binaries", fmt.Sprintf("core_arch_bin-%s-%s-go%s-%s", info.Version, sdkutils.GOARCH, goversion, tags)),
		Files: []string{
			"bin/flare",
			"core/plugin.so",
		},
	}

	if err := build.Run(); err != nil {
		panic(err)
	}
}

func BuildCore() {
	workdir := filepath.Join(sdkutils.PathTmpDir, "b/core", sdkutils.RandomStr(16))
	defer os.RemoveAll(workdir)

	if err := pkg.BuildTemplates(sdkutils.PathCoreDir); err != nil {
		panic(err)
	}

	if err := pkg.BuildSQLC(sdkutils.PathCoreDir); err != nil {
		panic(err)
	}

	if err := pkg.BuildPluginSo(sdkutils.PathCoreDir, workdir); err != nil {
		panic(err)
	}

	if err := pkg.BuildGlobalAssets(); err != nil {
		panic(err)
	}
}
