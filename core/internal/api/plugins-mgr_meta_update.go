//go:build !mono

package api

import (
	"core/utils/config"
	"core/utils/plugins"
	"fmt"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// RepinMetaRecordsToLatest refreshes every meta-bundle record to the bundle's
// latest cloud release (name, version, members). Meta bundles have no .so of their
// own — their members are ordinary store plugins that the system update already
// rebuilds against the new core. This only advances the bundle's METADATA so the
// Software Updates page stops showing the bundle as "update available" once the
// members it points to have been refreshed.
//
// It is called as a best-effort step during a system update (after the store
// plugins are staged): a failed lookup for one bundle is logged and skipped rather
// than aborting the whole update. Non-mono only.
func (self *PluginsMgr) RepinMetaRecordsToLatest() error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}

	for _, m := range cfg.MetaPlugins {
		rel, err := self.fetchStoreRelease(m.Package, "")
		if err != nil {
			self.CoreAPI.Logger().Error(fmt.Sprintf("RepinMetaRecordsToLatest: fetch latest %s: %v", m.Package, err))
			continue
		}
		if !rel.IsMeta {
			continue
		}

		memberPkgs := make([]string, 0, len(rel.Members))
		for _, member := range rel.Members {
			memberPkgs = append(memberPkgs, member.Package)
		}

		rec := sdkutils.MetaPlugin{
			Package: m.Package,
			Name:    rel.Name,
			Version: rel.Version,
			Members: memberPkgs,
		}
		if err := plugins.WriteMetaRecord(rec); err != nil {
			self.CoreAPI.Logger().Error(fmt.Sprintf("RepinMetaRecordsToLatest: save record %s: %v", m.Package, err))
			continue
		}
	}

	return nil
}
