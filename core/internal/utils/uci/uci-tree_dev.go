//go:build dev

package uci

import (
	"path/filepath"

	"github.com/digineo/go-uci"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

var treeRoot = filepath.Join(sdkutils.PathDataDir, "openwrt-files/etc/config")
var UciTree = uci.NewTree(treeRoot)
