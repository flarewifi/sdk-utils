package tools

import (
	"fmt"
	"os"
	"path/filepath"

	sdkplugin "sdk/api/plugin"

	sdkfs "github.com/flarehotspot/go-utils/fs"
	sdkruntime "github.com/flarehotspot/go-utils/runtime"
	sdkstr "github.com/flarehotspot/go-utils/strings"
)

func CreatePlugin(pack string, name string, desc string) {
	info := sdkplugin.PluginInfo{
		Name:        name,
		Package:     pack,
		Description: desc,
		Version:     "0.0.1",
	}

	goVersion := sdkruntime.GO_VERSION
	pluginDir := filepath.Join("plugins/local", pack)
	if sdkfs.Exists(pluginDir) {
		fmt.Printf("Plugin already exists at %s\n", pluginDir)
		os.Exit(1)
	}

	sdkfs.EnsureDir(pluginDir)

	modPath := filepath.Join(pluginDir, "go.mod")
	modUri := fmt.Sprintf("com.mydomain.%s", sdkstr.Rand(8))
	goMod := fmt.Sprintf("module %s\n\ngo %s", modUri, goVersion)
	if err := os.WriteFile(modPath, []byte(goMod), 0644); err != nil {
		panic(err)
	}

	pluginJson := filepath.Join(pluginDir, "plugin.json")
	if err := sdkfs.WriteJson(pluginJson, &info); err != nil {
		panic(err)
	}

	mainPath := filepath.Join(pluginDir, "main.go")

	goMain := `
package main

import (
    sdkplugin "sdk/api/plugin"
)

func main() {}

func Init(api sdkplugin.PluginApi) {
    // Your plugin code here
}
    `

	if err := os.WriteFile(mainPath, []byte(goMain), 0644); err != nil {
		panic(err)
	}

	gitIgnorePath := filepath.Join(pluginDir, ".gitignore")
	gitIgnore := `
.DS_Store
/node_modules
/resources/assets/dist
*.so
main_mono.go
*_templ.go
`
	if err := os.WriteFile(gitIgnorePath, []byte(gitIgnore), 0644); err != nil {
		panic(err)
	}

	CreateGoWorkspace()

	fmt.Printf("\n\nPlugin created at %s\n", pluginDir)
}
