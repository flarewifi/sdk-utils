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

	// Standard signature: func Init(api sdkapi.IPluginApi) error
	if initFn, ok := initSym.(func(sdkapi.IPluginApi) error); ok {
		if err := initFn(api); err != nil {
			return err
		}
		return nil
	}

	log.Println("Error: Init function has unexpected signature")
	return nil
}
