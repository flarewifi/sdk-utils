package tools

import (
	"core/internal/utils/pkg"
	"fmt"
	"os"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func CreatePlugin(pack string, name string, desc string) {
	coreInfo := pkg.GetCoreInfo()
	info := sdkutils.PluginInfo{
		Name:           name,
		Package:        pack,
		Description:    desc,
		Version:        "0.0.1",
		SystemPackages: []string{},
		SDK:            coreInfo.Version,
	}

	goVersion := sdkutils.GO_SHORT_VERSION
	pluginDir := filepath.Join("plugins/local", pack)
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
	if err := sdkutils.FsWriteJson(pluginJson, &info); err != nil {
		panic(err)
	}

	mainPath := filepath.Join(pluginDir, "main.go")

	goMain := `
package main

import (
	"fmt"
	"net/http"

	sdkapi "sdk/api"
)

func main() {}

func Init(api sdkapi.IPluginApi) {
	// Your plugin code here
	httpAPI := api.Http()
	pluginConfigAPI := api.Config().Plugin()
	adminRouter := httpAPI.HttpRouter().AdminRouter()

	// register the settings form
	if err := httpAPI.Forms().RegisterForm("settings", func(r *http.Request) sdkapi.HttpForm {

		// Return the form definition
		return sdkapi.HttpForm{
			CallbackRoute: "settings:save",
			SubmitLabel:   "Submit",
			Sections: []sdkapi.FormSection{
				{
					Name:  "general_configuration",
					Label: "General Configuration",
					Fields: []sdkapi.IFormField{
						sdkapi.FormTextField{
							Name:  "banner_text",
							Label: "Banner Text",
							ValueFn: func() string {
								b, err := pluginConfigAPI.Read("banner_text")
								if err != nil {
									return "This is the default banner text!"
								}
								return string(b)
							},
						},
					},
				},
			},
		}

	}); err != nil {
		api.Logger().Error(fmt.Sprintf("Failed to register settings form: %s", err))
		return
	}

	// Add a new route group to the admin router
	adminRouter.Group("/settings", func(subrouter sdkapi.IHttpRouterInstance) {

		subrouter.Get("/form", func(w http.ResponseWriter, r *http.Request) {
			// Get the form template
			formTemplate, err := httpAPI.Forms().GetFormTemplate("settings", r)
			if err != nil {
				httpAPI.HttpResponse().Error(w, r, err, http.StatusInternalServerError)
				return
			}

			httpAPI.HttpResponse().AdminView(w, r, sdkapi.ViewPage{PageContent: formTemplate})

		}).Name("settings:form")

		subrouter.Post("/save", func(w http.ResponseWriter, r *http.Request) {
			// Parse and validate the form input values
			form, err := httpAPI.Forms().ParseForm("settings", r)
			if err != nil {
				httpAPI.HttpResponse().Error(w, r, err, http.StatusInternalServerError)
				return
			}

			bannerText, err := form.GetStringValue("general_configuration", "banner_text")
			if err != nil {
				httpAPI.HttpResponse().Error(w, r, err, http.StatusInternalServerError)
				return
			}

			// Write the new value to the plugin configuration and send a success message
			pluginConfigAPI.Write("banner_text", []byte(bannerText))
			httpAPI.HttpResponse().FlashMsg(w, r, "Settings saved successfully", sdkapi.FlashMsgSuccess)
			httpAPI.HttpResponse().Redirect(w, r, "settings:form")

		}).Name("settings:save")
	})

	// Register navigation menu items
	httpAPI.Navs().AdminNavsFactory(func(r *http.Request) []sdkapi.AdminNavItemOpt {
		return []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategorySystem,
				Label:     "My Plugin",
				RouteName: "settings:form",
			},
		}
	})
}`

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

	CreateGoWorkspace()

	if err := pkg.ValidateSrcPath(pluginDir); err != nil {
		panic("Error validating newly created plugin: " + err.Error())
	}

	fmt.Printf("\n\nPlugin created at %s\nDone.\n", pluginDir)
}
