package main

import (
	"core/internal/utils/pkg"

	sdkpaths "github.com/flarehotspot/go-utils/paths"
)

func main() {
	pkg.LinkNodeModulesLib(sdkpaths.AppDir)
}
