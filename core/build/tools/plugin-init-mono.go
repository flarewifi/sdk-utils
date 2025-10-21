package tools

import (
	"core/internal/utils/plugins"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type PluginModule struct {
	PluginImportVar   string
	PluginModuleUri   string
	PluginPackageName string
}

func CreateMonoFiles() {
	CreateGoWorkspace()

	if err := plugins.BuildGlobalAssets(); err != nil {
		panic(err)
	}

	localDefs := plugins.LocalPluginSrcDefs()
	systemDefs := plugins.SystemPluginSrcDefs()

	pluginDirs := []string{filepath.Join(sdkutils.PathAppDir, "core")}
	for _, def := range append(systemDefs, localDefs...) {
		pluginDirs = append(pluginDirs, def.LocalPath)
	}

	for _, p := range pluginDirs {
		MakePluginMainMono(p)
	}

	MakePluginInitMono()
}

func MakePluginInitMono() {
	pluginPaths := []string{"core"}
	pluginDirs := []string{}

	localDefs := plugins.LocalPluginSrcDefs()
	systemDefs := plugins.SystemPluginSrcDefs()
	for _, def := range append(systemDefs, localDefs...) {
		pluginDirs = append(pluginDirs, def.LocalPath)
	}

	pluginPaths = append(pluginPaths, pluginDirs...)
	coreInfo := plugins.GetCoreInfo()

	pluginMods := []PluginModule{}
	for _, dir := range pluginDirs {
		modVar := sdkutils.Slugify(filepath.Base(dir), "_")
		modPath := getGoModule(dir)
		pkgName := getPackage(dir)
		mod := PluginModule{modVar, modPath, pkgName}
		pluginMods = append(pluginMods, mod)
	}

	importModules := ""
	for _, mod := range pluginMods {
		importModules += fmt.Sprintf("\n\t"+`%s "%s"`, mod.PluginImportVar, mod.PluginModuleUri)
	}

	pluginSwitchCases := ""
	for _, mod := range pluginMods {
		pluginSwitchCases += fmt.Sprintf("\n\t\tcase \"%s\":\n\t\t\t%s.Init(p)", mod.PluginPackageName, mod.PluginImportVar)
	}

	pluginMonoInit := fmt.Sprintf(`//go:build mono

%s

package api
import (
    "log"
    %s
)

func (p *PluginApi) Init() error {
    info := p.Info()
    switch info.Package {
        case "%s":
            log.Println("core package, skipping plugin.Init()...")
%s
        default:
            log.Println("Unable to load plugin: " + p.dir)
    }
    return nil
}`, AUTO_GENERATED_HEADER, importModules, coreInfo.Package, pluginSwitchCases)

	pluginInitMonoPath := filepath.Join("core/internal/api/plugin-init_mono.go")

	var pluginInitMonoContent string
	if b, err := os.ReadFile(pluginInitMonoPath); err == nil {
		pluginInitMonoContent = string(b)
	}

	if pluginMonoInit != pluginInitMonoContent {
		err := os.WriteFile(pluginInitMonoPath, []byte(pluginMonoInit), 0644)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println(pluginInitMonoPath, "has been created.")
}

func getGoModule(pluginDir string) string {
	goModFile := filepath.Join(pluginDir, "go.mod")
	modContent, err := sdkutils.FsReadFile(goModFile)
	if err != nil {
		panic(err)
	}

	regx := regexp.MustCompile(`module\s+([\w\/.-]+)`)
	matches := regx.FindStringSubmatch(string(modContent))
	if len(matches) > 0 && len(matches[0]) > 0 {
		return strings.Split(matches[0], " ")[1]
	}

	panic("Error: go.mod file does not contain module name")
}

func getPackage(pluginDir string) string {
	info, err := sdkutils.GetPluginInfoFromPath(pluginDir)
	if err != nil {
		panic(err)
	}
	return info.Package
}
