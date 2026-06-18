//go:build !mono

package api

import (
	"fmt"
	"slices"
)

// LoadPlugin loads a plugin (maps its .so and resolves Init) and tracks it, but
// does NOT run its Init. The boot path uses this so plugins are loaded offline at
// boot while their Init is deferred until any internet-dependent provisioning has
// run (see boot.InitLoadedPlugins / ProvisionInstalledPlugins). Already-tracked
// plugins are a no-op.
func (self *PluginsMgr) LoadPlugin(p *PluginApi) error {
	if slices.Contains(self.plugins, p) {
		return nil
	}
	if err := self.loadPluginApi(p); err != nil {
		return err
	}
	self.plugins = append(self.plugins, p)
	return nil
}

// RegisterPlugin loads AND initializes a plugin immediately, then tracks it. This
// is the runtime path (dashboard install, recovery): the device is already online
// and the plugin's system_packages/preinstall have run inline, so there is nothing
// to wait for. The plugin is tracked only after Init succeeds, preserving the
// previous "not registered on init failure" behavior.
func (self *PluginsMgr) RegisterPlugin(p *PluginApi) error {
	if slices.Contains(self.plugins, p) {
		return nil
	}
	if err := self.loadPluginApi(p); err != nil {
		return err
	}
	if err := p.RunInit(); err != nil {
		return fmt.Errorf("%w: Error initializing plugin: %v", err, p.Dir())
	}
	self.plugins = append(self.plugins, p)
	return nil
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// loadPluginApi wires the plugin's API, loads its assets manifest, and maps its
// .so (resolving Init). It does not invoke Init and does not track the plugin.
func (self *PluginsMgr) loadPluginApi(p *PluginApi) error {
	p.Initialize(self.CoreAPI)
	p.LoadAssetsManifest()
	if err := p.Load(); err != nil {
		return err
	}
	return nil
}
