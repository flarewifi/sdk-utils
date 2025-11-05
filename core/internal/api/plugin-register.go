//go:build !mono

package api

import (
	"fmt"
	"log"
	"slices"
)

func (self *PluginsMgr) RegisterPlugin(p *PluginApi) error {

	exists := slices.Contains(self.plugins, p)
	if !exists {
		p.Initialize(self.CoreAPI)
		p.LoadAssetsManifest()

		err := p.Init()
		if err != nil {
			log.Println("Error initializing plugin: "+p.Dir(), err)
			// TODO: set plugin as broken
			return fmt.Errorf("%w: Error initializing plugin: %v", err, p.Dir())
		}

		self.plugins = append(self.plugins, p)
	}

	return nil
}
