//go:build !mono

package api

import (
	"fmt"
	"log"
)

func (self *PluginsMgr) RegisterPlugin(p *PluginApi) error {
	if p.Info().Package != self.CoreAPI.Info().Package {
		err := p.Init()
		if err != nil {
			log.Println("Error initializing plugin: "+p.Dir(), err)
			// TODO: set plugin as broken
			return fmt.Errorf("%w: Error initializing plugin: %v", err, p.Dir())
		}
	}

	p.Initialize(self.CoreAPI)
	p.LoadAssetsManifest()
	self.plugins = append(self.plugins, p)

	return nil
}
