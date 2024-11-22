package tools

import (
	"core/env"
	"core/internal/utils/pkg"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	sdkpaths "github.com/flarehotspot/go-utils/paths"
	sdkruntime "github.com/flarehotspot/go-utils/runtime"
	sdkstr "github.com/flarehotspot/go-utils/strings"
)

func BuildCoreBins() {
	BuildFlareCLI()
	BuildCore()

	goversion := sdkruntime.GO_VERSION
	tags := sdkstr.Slugify(env.BuildTags, "-")

	build := &BuildOutput{
		OutputDirName: filepath.Join("core-binaries", fmt.Sprintf("%s-%s-go%s-%s", pkg.CoreInfo().Version, sdkruntime.GOARCH, goversion, tags)),
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
	workdir := filepath.Join(sdkpaths.TmpDir, "b/core", sdkstr.Rand(16))
	defer os.RemoveAll(workdir)

	InstallSqlc()

	cmd := exec.Command(sdkpaths.SqlcBin, "generate")
	cmd.Dir = sdkpaths.AppDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}

	if err := pkg.BuildTemplates(sdkpaths.CoreDir); err != nil {
		panic(err)
	}

	if err := pkg.BuildPluginSo(sdkpaths.CoreDir, workdir); err != nil {
		panic(err)
	}

	if err := pkg.BuildGlobalAssets(); err != nil {
		panic(err)
	}
}
