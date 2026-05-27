//go:build !mono && !dev

package server

import (
	"log"
	"path/filepath"
	"plugin"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func Server() {
	corePath := filepath.Join(sdkutils.PathAppDir, "core/plugin.so")
	p, err := plugin.Open(corePath)
	if err != nil {
		log.Println("Error loading core plugin:", err)
		panic(err)
	}
	symInit, _ := p.Lookup("Init")
	initFn := symInit.(func())
	initFn()
}
