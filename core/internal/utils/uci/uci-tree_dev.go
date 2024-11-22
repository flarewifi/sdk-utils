//go:build dev

package uci

import (
	"path/filepath"

	"github.com/digineo/go-uci"

	paths "github.com/flarehotspot/go-utils/paths"
)

var treeRoot = filepath.Join(paths.AppDir, "openwrt-files/etc/config")
var UciTree = uci.NewTree(treeRoot)
