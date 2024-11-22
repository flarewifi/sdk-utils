//go:build !dev

package uci

import "github.com/digineo/go-uci"

var UciTree = uci.NewTree("/etc/config")
