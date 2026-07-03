//go:build !mono

package api

import (
	"core/utils/config"
	"core/utils/plugins"
	"fmt"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// RepinMetaRecordsToLatest refreshes every meta-bundle record to the bundle's
// current cloud membership (name + member package list). Meta bundles have no .so
// of their own — their members are ordinary store plugins that the system update
// already updates to their own latest independently. This advances the bundle's
// metadata so the local record reflects the current member set (for the uninstall
// cascade and coverage display).
//
// Members dropped from a bundle are NOT uninstalled here: a dropped member simply
// loses bundle coverage. If it is free or separately purchased it keeps working and
// updates on its own; if it was paid-only-through-the-bundle it is caught at the
// next boot by ValidateStorePlugins (disabled + admin-notified, "Buy Now" in the
// store) rather than being removed. Best-effort: a failed lookup for one bundle is
// logged and skipped rather than aborting the update. Non-mono only.
func (self *PluginsMgr) RepinMetaRecordsToLatest() error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}

	for _, m := range cfg.MetaPlugins {
		rel, err := self.fetchStoreRelease(m.Package, "", "")
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
			self.CoreAPI.Logger().Error(fmt.Sprintf("RepinMetaRecordsToLatest: save record %s: %v", rec.Package, err))
		}
	}

	return nil
}
