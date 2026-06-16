package api

import (
	"fmt"

	"core/utils/config"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// Meta-plugin bundle handling. A meta plugin is a named bundle of other plugins;
// it has no plugin.so of its own. Its bundle -> members mapping lives in
// data/config/plugins.json (cfg.MetaPlugins), which is the single source of truth
// for membership and uninstall cascades. The INSTALL side is in plugins-mgr.go
// (InstallPlugin -> installStoreMeta); this file holds the uninstall cascade and
// the membership queries so all meta logic lives in core's PluginsMgr rather than
// in the store UI plugin.

// uninstallMeta removes a meta bundle record and cascades to its members: any
// member left owned by no remaining meta and not installed standalone is marked
// for removal (applied on the next restart). Ownership is recomputed against the
// updated config, so members shared with another bundle (or installed on their
// own) are preserved. Symmetric with InstallPlugin's meta-bundle expansion.
//
// This is an internal helper: callers use UninstallPlugin, which routes a meta
// bundle here via isMetaPlugin so a single entry point handles both regular and
// meta plugins.
func (self *PluginsMgr) uninstallMeta(pkg string) error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}

	var removed sdkutils.MetaPlugin
	found := false
	kept := make([]sdkutils.MetaPlugin, 0, len(cfg.MetaPlugins))
	for _, m := range cfg.MetaPlugins {
		if m.Package == pkg {
			removed = m
			found = true
			continue
		}
		kept = append(kept, m)
	}
	if !found {
		return fmt.Errorf("meta plugin not found: %s", pkg)
	}

	cfg.MetaPlugins = kept
	if err := config.WritePluginsConfig(cfg); err != nil {
		return err
	}

	// cfg now excludes the removed bundle, so metaOwnersOf reflects remaining
	// ownership. A member kept by another bundle or installed standalone survives.
	for _, member := range removed.Members {
		if len(metaOwnersOf(cfg, member)) == 0 && !isStandalone(cfg, member) {
			if err := self.UninstallPlugin(member); err != nil {
				self.CoreAPI.Logger().Error(fmt.Sprintf("uninstallMeta: uninstall member %s: %v", member, err))
			}
		}
	}

	return nil
}

// MetaPlugins returns all installed meta-plugin bundle records.
func (self *PluginsMgr) MetaPlugins() ([]sdkutils.MetaPlugin, error) {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return nil, err
	}
	return cfg.MetaPlugins, nil
}

// MetaMembership reports which installed meta bundles own pkg and whether it
// should be treated as a standalone install. A plugin installed on its own, or
// owned by no meta, is standalone. When the plugins config cannot be read it
// returns ([]string{}, true) — the safe default (no owners, treated standalone),
// so a transient read failure never hides a plugin or implies meta ownership.
func (self *PluginsMgr) MetaMembership(pkg string) (owners []string, standalone bool) {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return []string{}, true
	}
	owners = metaOwnersOf(cfg, pkg)
	return owners, isStandalone(cfg, pkg) || len(owners) == 0
}

// isMetaPlugin reports whether pkg is an installed meta bundle (has a record in
// cfg.MetaPlugins). Used by Uninstall to route a bundle to the cascade.
func (self *PluginsMgr) isMetaPlugin(pkg string) bool {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return false
	}
	for _, m := range cfg.MetaPlugins {
		if m.Package == pkg {
			return true
		}
	}
	return false
}

// metaOwnersOf returns the packages of meta bundles that include pkg as a member.
func metaOwnersOf(cfg sdkutils.PluginsConfig, pkg string) []string {
	var owners []string
	for _, m := range cfg.MetaPlugins {
		for _, member := range m.Members {
			if member == pkg {
				owners = append(owners, m.Package)
				break
			}
		}
	}
	return owners
}

// isStandalone reports whether pkg was installed on its own (its metadata flag),
// defaulting to false when no metadata exists.
func isStandalone(cfg sdkutils.PluginsConfig, pkg string) bool {
	for _, md := range cfg.Metadata {
		if md.Package == pkg {
			return md.Standalone
		}
	}
	return false
}
