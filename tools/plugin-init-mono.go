package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"tools/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type PluginModule struct {
	PluginImportVar   string
	PluginModuleUri   string
	PluginPackageName string
}

func CreateMonoPluginInit() {
	CreateGoWorkspace()

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
	pluginPaths := []string{}
	pluginDirs := []string{}

	localDefs := plugins.LocalPluginSrcDefs()
	systemDefs := plugins.SystemPluginSrcDefs()
	for _, def := range append(systemDefs, localDefs...) {
		pluginDirs = append(pluginDirs, def.LocalPath)
	}

	pluginPaths = append(pluginPaths, pluginDirs...)

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

	pluginInitCodes := `
	// Load core plugin
	info, err = sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir)
	if err != nil {
		fmt.Println("Error loading core plugin: " + err.Error())
	} else {
		// Load plugin
		if err := migrate.MigrateUp(self.db.DB, sdkutils.PathCoreDir); err != nil {
			fmt.Println("failed to run migrations for core plugin:" + err.Error())
		}
		api = coreAPI
		api.Initialize(coreAPI)
		api.LoadAssetsManifest()
		self.plugins = append(self.plugins, api)
		fmt.Println("Loaded core plugin.")
	}
`
	for _, mod := range pluginMods {
		pkg := mod.PluginPackageName
		importVar := mod.PluginImportVar
		pluginInitCodes += fmt.Sprintf(`

	// Loading plugin
	pkg = "%s"
	fmt.Println("Loading mono plugin:" + pkg)

	pluginDir = filepath.Join(sdkutils.PathPluginInstallDir, pkg)
	info, err = sdkutils.GetPluginInfoFromPath(pluginDir)
	if err != nil {
		fmt.Println("Error getting plugin info from path: " + err.Error())
	} else {
		// Run migrations
		if err := migrate.MigrateUp(self.db.DB, pluginDir); err != nil {
			fmt.Println("Warning: failed to apply migrations for plugin :", pkg)
		}
		// Load plugin
		api = NewPluginApi(pluginDir, info, g.GlobalAssets, self, self.trfkMgr)
		%s.Init(api)
		api.Initialize(coreAPI)
		api.LoadAssetsManifest()
		self.plugins = append(self.plugins, api)
		fmt.Println("Loaded mono plugin:" + pkg)
	}

		`, pkg, importVar)
	}

	pluginMonoInit := fmt.Sprintf(`//go:build mono

%s

package api
import (
	"fmt"
	"path/filepath"

	"tools/migrate"
	sdkutils "github.com/flarehotspot/sdk-utils"

    %s
)

func (self *PluginsMgr) RegisterPlugin(p *PluginApi) error {
	return nil
}

func (self *PluginsMgr) LoadMonoPlugins(g *CoreGlobals) {
	coreAPI := self.CoreAPI

	var pkg, pluginDir string
	var api *PluginApi
	var info sdkutils.PluginInfo
	var err error

	%s
}

`, AUTO_GENERATED_HEADER, importModules, pluginInitCodes)

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
