//go:build !mono

package main

import (
	sdkapi "sdk/api"

	"com.flarego.default-theme/app"
	"com.flarego.default-theme/app/themes"
)

func main() {}

func Init(api sdkapi.IPluginApi) {
	app.SetupRoutes(api)
	themes.SetPortalTheme(api)
	themes.SetAdminTheme(api)
}
