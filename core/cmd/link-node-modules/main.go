package main

import (
	"core/internal/utils/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func main() {
	plugins.LinkNodeModulesLib(sdkutils.PathAppDir)
}
