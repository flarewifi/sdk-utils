//go:build !mono && !dev

package server

import (
	"path/filepath"
	"plugin"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func Server() {
	corePath := filepath.Join(sdkutils.PathAppDir, "core/plugin.so")
	p, err := plugin.Open(corePath)
	if err != nil {
		panic(err)
	}
	symInit, _ := p.Lookup("Init")
	initFn := symInit.(func())
	initFn()
}
