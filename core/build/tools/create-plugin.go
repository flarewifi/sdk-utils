package tools

import (
	"core/internal/utils/plugins"
	"fmt"
	"os"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func CreatePlugin(pack string, name string, desc string) {
	coreInfo := plugins.GetCoreInfo()
	info := sdkutils.PluginInfo{
		Name:           name,
		Package:        pack,
		Description:    desc,
		Version:        "0.0.1",
		SystemPackages: []string{},
		SDK:            coreInfo.Version,
	}

	goVersion := sdkutils.GO_SHORT_VERSION
	pluginDir := filepath.Join(sdkutils.PathPluginLocalDir, pack)
	if sdkutils.FsExists(pluginDir) {
		fmt.Printf("Plugin already exists at %s\n", pluginDir)
		os.Exit(1)
	}

	sdkutils.FsEnsureDir(pluginDir)

	modPath := filepath.Join(pluginDir, "go.mod")
	modUri := fmt.Sprintf("com.mydomain.%s", sdkutils.RandomStr(8))
	goMod := fmt.Sprintf(`module %s
go %s

require (
	github.com/a-h/templ v0.2.793
)
`, modUri, goVersion)
	if err := os.WriteFile(modPath, []byte(goMod), 0644); err != nil {
		panic(err)
	}

	pluginJson := filepath.Join(pluginDir, "plugin.json")
	if err := sdkutils.JsonWrite(pluginJson, &info); err != nil {
		panic(err)
	}

	mainPath := filepath.Join(pluginDir, "main.go")

	goMain := fmt.Sprintf(`
package main

import (
	"net/http"

	views "%s/resources/views"
	sdkapi "sdk/api"
)

func main() {}

func Init(api sdkapi.IPluginApi) {
	// Your plugin code here
	adminRouter := api.Http().Router().AdminRouter()

	adminRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
		homePage := views.Home()
		api.Http().Response().AdminView(w, r, sdkapi.ViewPage{
			PageContent: homePage,
		})
	}).Name("home.index")

	api.Http().Navs().AdminNavsFactory(func(r *http.Request) []sdkapi.AdminNavItemOpt {
		return []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     "My Plugin",
				RouteName: "home.index",
				Keywords:  []string{"sample", "home"},
			},
		}
	})
}
`, modUri)

	if err := os.WriteFile(mainPath, []byte(goMain), 0644); err != nil {
		panic(err)
	}

	MakePluginMainMono(pluginDir)

	home := `
package views

templ Home() {
	<h1>Welcome to your new plugin!</h1>
}
	`

	homeTempl := filepath.Join(pluginDir, "resources/views/home.templ")
	if err := os.MkdirAll(filepath.Dir(homeTempl), sdkutils.PermDir); err != nil {
		panic(err)
	}
	if err := os.WriteFile(homeTempl, []byte(home), sdkutils.PermFile); err != nil {
		panic(err)
	}

	gitIgnorePath := filepath.Join(pluginDir, ".gitignore")
	gitIgnore := `
.DS_Store
/node_modules
/resources/assets/dist
*.so
`
	if err := os.WriteFile(gitIgnorePath, []byte(gitIgnore), 0644); err != nil {
		panic(err)
	}

	licenseFile := filepath.Join(pluginDir, "LICENSE.txt")
	licenseTxt := `# No License Chosen

This software does not currently have a license.

By default, all rights are reserved. This means:
- You may view the code.
- You may not use, modify, or distribute this software for any purpose without explicit written permission from the copyright holder.

The license for this software is still under consideration and will be added in the future. Until then, please contact [YOUR CONTACT INFORMATION] for any inquiries about usage or licensing.
`
	if err := os.WriteFile(licenseFile, []byte(licenseTxt), sdkutils.PermFile); err != nil {
		panic(err)
	}

	sqlcYaml := `---
version: '2'
sql:
  - engine: postgresql
    queries: [resources/queries]
    schema: [resources/migrations]
    gen:
      go:
        package: queries
        out: db/queries
        sql_package: pgx/v5
`

	sqlcPath := filepath.Join(pluginDir, "sqlc.yaml")
	if err := sdkutils.FsWriteFile(sqlcPath, []byte(sqlcYaml)); err != nil {
		panic(err)
	}

	// Create default images directory
	pubkeep := filepath.Join(pluginDir, "resources/assets/plublic/.keep")
	if err := os.MkdirAll(filepath.Dir(pubkeep), sdkutils.PermDir); err != nil {
		panic(err)
	}

	if _, err := os.Create(pubkeep); err != nil {
		panic(err)
	}

	CreateGoWorkspace()

	if err := plugins.ValidateSrcPath(pluginDir); err != nil {
		panic("Error validating newly created plugin: " + err.Error())
	}

	fmt.Printf("\n\nPlugin created at %s\nDone.\n", pluginDir)
}
