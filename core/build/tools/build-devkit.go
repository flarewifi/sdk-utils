package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"core/env"
	"core/internal/utils/plugins"
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"

	"github.com/goccy/go-json"
)

var (
	devkitReleaseDir string
	devkitFiles      = []string{
		".go-version",
		"bin/flare",
		"config/.defaults",
		"core/go.mod",
		"core/plugin.so",
		"core/plugin.json",
		"core/resources",
		"go.work.default",
		"main/go.mod",
		"plugins/system",
		"scripts",
		"sdk",
	}
)

func init() {
	goversion := sdkutils.GO_VERSION
	tags := sdkutils.Slugify(env.BuildTags, "-")
	devkitReleaseDir = filepath.Join(sdkutils.PathAppDir, "output/devkit", fmt.Sprintf("devkit-%s-%s-go%s-%s", plugins.GetCoreInfo().Version, runtime.GOARCH, goversion, tags))
}

func CreateDevkit() {
	// Clean up output path
	if err := sdkutils.FsEmptyDir(filepath.Dir(devkitReleaseDir)); err != nil {
		panic(err)
	}

	// Build the bin/flare cli
	BuildFlareCLI()

	// Build core/plugin.so
	BuildCore()

	// Copy devkit files
	for _, entry := range devkitFiles {
		srcPath := filepath.Join(sdkutils.PathAppDir, entry)
		destPath := filepath.Join(devkitReleaseDir, entry)
		fmt.Println("Copying: ", sdkutils.StripRootPath(srcPath), " -> ", sdkutils.StripRootPath(destPath))

		if err := sdkutils.FsCopy(srcPath, destPath); err != nil {
			panic(err)
		}
	}

	// Copy extra devkit files to the release directory
	extrasPath := filepath.Join(sdkutils.PathAppDir, "core/build/devkit/extras")
	fmt.Printf("Copying:  %s -> %s\n", sdkutils.StripRootPath(extrasPath), sdkutils.StripRootPath(devkitReleaseDir))
	err := sdkutils.FsCopyDir(extrasPath, devkitReleaseDir, nil)
	if err != nil {
		panic(err)
	}

	// Copy default go.work
	goWorkDefaultPath := filepath.Join(sdkutils.PathAppDir, "go.work.default")
	goWorkPath := filepath.Join(devkitReleaseDir, "go.work")
	fmt.Println("Copying: ", sdkutils.StripRootPath(goWorkDefaultPath), " -> ", sdkutils.StripRootPath(goWorkPath))
	if err := sdkutils.FsCopyFile(goWorkDefaultPath, goWorkPath); err != nil {
		panic(err)
	}

	// Generate default application config
	appConfigFile := filepath.Join(devkitReleaseDir, "config/application.json")
	appConfig := sdkapi.AppCfg{
		Lang:     "en",
		Currency: "php",
		Secret:   sdkutils.RandomStr(16),
	}

	b, err := json.MarshalIndent(appConfig, "", "  ")
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile(appConfigFile, b, 0644); err != nil {
		panic(err)
	}

	fmt.Println("Application config created: ", sdkutils.StripRootPath(appConfigFile))

	// Clean up core template files
	var templateFiles []string
	if err := sdkutils.FsListFiles(filepath.Join(devkitReleaseDir, "core/resources/views"), &templateFiles, true); err != nil {
		panic(err)
	}

	for _, file := range templateFiles {
		if filepath.Ext(file) == ".templ" || strings.HasSuffix(file, "_templ.go") {
			fmt.Println("Removing: ", sdkutils.StripRootPath(file))
			if err := os.Remove(file); err != nil {
				panic(err)
			}
		}
	}

	// Compress devkit release files
	file := filepath.Base(devkitReleaseDir) + ".zip"
	dir := filepath.Dir(devkitReleaseDir)
	filepath := filepath.Join(dir, file)
	err = sdkutils.CompressZip(devkitReleaseDir, filepath)
	if err != nil {
		panic(err)
	}

	fmt.Println("Devkit created: ", sdkutils.StripRootPath(filepath))
}
