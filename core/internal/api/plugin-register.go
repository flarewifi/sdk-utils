//go:build !mono

package api

import (
	"fmt"
	"slices"
)

func (self *PluginsMgr) RegisterPlugin(p *PluginApi) error {

	exists := slices.Contains(self.plugins, p)
	if !exists {
		p.Initialize(self.CoreAPI)
		p.LoadAssetsManifest()

		err := p.Init()
		if err != nil {
			return fmt.Errorf("%w: Error initializing plugin: %v", err, p.Dir())
		}

		self.plugins = append(self.plugins, p)
	}

	return nil
}
