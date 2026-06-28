//go:build !mono

package main

import (
	sdkapi "sdk/api"

	"com.flarego.devkit/app/themes"
)

func main() {}

func Init(api sdkapi.IPluginApi) error {
	return themes.Setup(api)
}
