package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"core/env"
	"core/internal/utils/pkg"
	sdkcfg "sdk/api/config"

	sdkfs "github.com/flarehotspot/go-utils/fs"
	sdkpaths "github.com/flarehotspot/go-utils/paths"
	sdkruntime "github.com/flarehotspot/go-utils/runtime"
	sdkstr "github.com/flarehotspot/go-utils/strings"
	sdkzip "github.com/flarehotspot/go-utils/zip"

	"github.com/goccy/go-json"
)

var (
	devkitReleaseDir string
	devkitFiles      = []string{
		".tmp/go",
		".go-version",
		"tools.sh",
		"bin",
		"config/.defaults",
		"core/go.mod",
		"core/plugin.so",
		"core/plugin.json",
		"core/resources",
		"plugins/system",
		"main/go.mod",
		"sdk",
	}
)

func init() {
	goversion := sdkruntime.GO_VERSION
	tags := sdkstr.Slugify(env.BuildTags, "-")
	devkitReleaseDir = filepath.Join(sdkpaths.AppDir, "output/devkit", fmt.Sprintf("devkit-%s-%s-go%s-%s", pkg.CoreInfo().Version, runtime.GOARCH, goversion, tags))
}

func CreateDevkit() {
	PrepareCleanup()
	InstallGo(".tmp/go")
	BuildFlareCLI()
	BuildCore()
	CopyDevkitFiles()
	CopyDevkitExtras()
	CopyDefaultWorksapce()
	CreateApplicationConfig()
	ZipDevkitRelease()
}

func CreateApplicationConfig() {
	cfgPath := filepath.Join(devkitReleaseDir, "config/application.json")
	appcfg := sdkcfg.AppCfg{
		Lang:     "en",
		Currency: "php",
		Secret:   sdkstr.Rand(16),
	}

	b, err := json.MarshalIndent(appcfg, "", "  ")
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile(cfgPath, b, 0644); err != nil {
		panic(err)
	}

	fmt.Println("Application config created: ", sdkpaths.StripRoot(cfgPath))
}

func CopyDevkitFiles() {
	for _, entry := range devkitFiles {
		srcPath := filepath.Join(sdkpaths.AppDir, entry)
		destPath := filepath.Join(devkitReleaseDir, entry)
		fmt.Println("Copying: ", sdkpaths.StripRoot(srcPath), " -> ", sdkpaths.StripRoot(destPath))

		if err := sdkfs.Copy(srcPath, destPath); err != nil {
			panic(err)
		}
	}
}

func CopyDevkitExtras() {
	extrasPath := filepath.Join(sdkpaths.AppDir, "core/build/devkit/extras")
	fmt.Printf("Copying:  %s -> %s\n", sdkpaths.StripRoot(extrasPath), sdkpaths.StripRoot(devkitReleaseDir))
	err := sdkfs.CopyDir(extrasPath, devkitReleaseDir, nil)
	if err != nil {
		panic(err)
	}
}

func CopyDefaultWorksapce() {
	dst := filepath.Join(devkitReleaseDir, "go.work")
	def := "go.work.default"
	fmt.Println("Copying: ", sdkpaths.StripRoot(def), " -> ", sdkpaths.StripRoot(dst))
	sdkfs.CopyFile(def, dst)
}

func ZipDevkitRelease() {
	basename := filepath.Base(devkitReleaseDir) + ".zip"
	dir := filepath.Dir(devkitReleaseDir)
	zipFile := filepath.Join(dir, basename)
	err := sdkzip.Zip(devkitReleaseDir, zipFile)
	if err != nil {
		panic(err)
	}
	fmt.Println("Devkit created: ", sdkpaths.StripRoot(zipFile))
}

func PrepareCleanup() {
	dirsToRemove := []string{filepath.Dir(devkitReleaseDir)}
	for _, dir := range dirsToRemove {
		fmt.Println("Removing: ", filepath.Join(sdkpaths.AppDir, dir))
		err := os.RemoveAll(filepath.Join(sdkpaths.AppDir, dir))
		if err != nil {
			fmt.Println("Error removing: ", err)
			panic(err)
		}
	}
}
