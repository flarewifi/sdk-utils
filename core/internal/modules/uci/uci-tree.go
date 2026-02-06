package uci

import "github.com/digineo/go-uci"

// UciTree is the global UCI configuration tree.
// Path /etc/config is used in both development (mounted via docker-compose
// from ./openwrt-files/etc/config) and production (native OpenWRT path).
var UciTree = uci.NewTree("/etc/config")
