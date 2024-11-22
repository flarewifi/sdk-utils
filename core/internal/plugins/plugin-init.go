//go:build !mono

package plugins

import (
	"log"
	"path/filepath"
	"plugin"

	"sdk/api/plugin"
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

	initFn := initSym.(func(sdkplugin.IPluginApi))
	initFn(api)

	return nil
}
