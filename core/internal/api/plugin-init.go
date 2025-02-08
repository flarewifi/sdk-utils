//go:build !mono

package api

import (
	"log"
	"path/filepath"
	"plugin"

	sdkapi "sdk/api"
)

func (api *PluginApi) Init() error {
	pluginLib := filepath.Join(api.dir, "plugin.so")
	log.Println("Opening ", pluginLib)
	p, err := plugin.Open(pluginLib)
	if err != nil {
		return err
	}

	log.Println("Loaded ", pluginLib, "successfully")
	initSym, err := p.Lookup("Init")
	if err != nil {
		return err
	}

	initFn := initSym.(func(sdkapi.IPluginApi))
	initFn(api)

	return nil
}
