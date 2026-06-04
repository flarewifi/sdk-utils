//go:build !mono

package api

import (
	"path/filepath"
	"plugin"

	sdkapi "sdk/api"
)

func (api *PluginApi) Init() error {
	pluginLib := filepath.Join(api.dir, "plugin.so")
	p, err := plugin.Open(pluginLib)
	if err != nil {
		return err
	}

	initSym, err := p.Lookup("Init")
	if err != nil {
		return err
	}

	if initFn, ok := initSym.(func(sdkapi.IPluginApi) error); ok {
		if err := initFn(api); err != nil {
			return err
		}
		return nil
	}

	return nil
}
