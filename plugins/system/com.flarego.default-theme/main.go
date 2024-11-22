//go:build !mono

package main

import (
	"com.flarego.default-theme/app/themes"
	"com.flarego.default-theme/app"
	plugin "sdk/api/plugin"
)

func main() {}

func Init(api plugin.IPluginApi) {
    app.SetupRoutes(api)
	themes.SetPortalTheme(api)
    themes.SetAdminTheme(api)
}
