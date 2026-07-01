//go:build !mono

package api

import (
	"core/utils/config"
	"core/utils/plugins"
	"fmt"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// RepinMetaRecordsToLatest refreshes every meta-bundle record to the bundle's
// latest cloud release (name, version, members) and UNINSTALLS any member dropped
// from a bundle that, after the re-pin, is owned by no remaining bundle and not
// installed standalone (the same orphan cascade as uninstallMeta). Meta bundles
// have no .so of their own — their members are ordinary store plugins that the
// system update already rebuilds against the new core. This advances the bundle's
// METADATA so the Software Updates page stops showing it as "update available", and
// makes "remove a member in the next release" actually take effect on-device.
//
// confirmRemoval, when non-nil, is invoked with the planned orphan removals BEFORE
// anything is written; returning false aborts with ErrMetaMemberRemovalCancelled
// and applies nothing (the update flow discards the staged set). A nil callback
// applies removals without prompting.
//
// It is called as a best-effort step during a system update (after the store
// plugins are staged): a failed lookup for one bundle is logged and skipped rather
// than aborting the whole update. Non-mono only.
func (self *PluginsMgr) RepinMetaRecordsToLatest(confirmRemoval func([]MetaMemberRemoval) bool) error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}

	// First pass (read-only): resolve each bundle's latest release into a new record.
	// A failed fetch keeps the old record (and its members), so it must not count
	// toward orphan decisions below.
	repins := make(map[string]sdkutils.MetaPlugin, len(cfg.MetaPlugins))
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
		repins[m.Package] = sdkutils.MetaPlugin{
			Package: m.Package,
			Name:    rel.Name,
			Version: rel.Version,
			Members: memberPkgs,
		}
	}

	// Prospective config: the bundle set as it WILL look after the re-pin (new member
	// lists for successfully-fetched bundles, unchanged for the rest). Orphan status
	// is computed against this so a member kept by another bundle survives.
	prospective := cfg
	prospective.MetaPlugins = make([]sdkutils.MetaPlugin, 0, len(cfg.MetaPlugins))
	for _, m := range cfg.MetaPlugins {
		if rec, ok := repins[m.Package]; ok {
			prospective.MetaPlugins = append(prospective.MetaPlugins, rec)
		} else {
			prospective.MetaPlugins = append(prospective.MetaPlugins, m)
		}
	}

	// Planned removals: members dropped by a re-pinned bundle that end up orphaned.
	var removals []MetaMemberRemoval
	seen := make(map[string]bool)
	for _, m := range cfg.MetaPlugins {
		rec, ok := repins[m.Package]
		if !ok {
			continue
		}
		for _, member := range droppedMembers(m.Members, rec.Members) {
			if seen[member] {
				continue
			}
			if len(metaOwnersOf(prospective, member)) == 0 && !isStandalone(prospective, member) {
				removals = append(removals, MetaMemberRemoval{MetaPackage: m.Package, MetaName: rec.Name, Member: member})
				seen[member] = true
			}
		}
	}

	// Confirmation gate: nothing is written until the admin approves the removals.
	if len(removals) > 0 && confirmRemoval != nil {
		if !confirmRemoval(removals) {
			return ErrMetaMemberRemovalCancelled
		}
	}

	// Apply: write the new records first so the config reflects the new ownership,
	// then uninstall the orphaned dropped members (marked for removal on next boot).
	for _, rec := range repins {
		if err := plugins.WriteMetaRecord(rec); err != nil {
			self.CoreAPI.Logger().Error(fmt.Sprintf("RepinMetaRecordsToLatest: save record %s: %v", rec.Package, err))
			continue
		}
	}
	for _, rm := range removals {
		if err := self.UninstallPlugin(rm.Member); err != nil {
			self.CoreAPI.Logger().Error(fmt.Sprintf("RepinMetaRecordsToLatest: uninstall dropped member %s: %v", rm.Member, err))
		}
	}

	return nil
}

// droppedMembers returns the packages present in old but absent from current.
func droppedMembers(old, current []string) []string {
	keep := make(map[string]bool, len(current))
	for _, p := range current {
		keep[p] = true
	}
	var dropped []string
	for _, p := range old {
		if !keep[p] {
			dropped = append(dropped, p)
		}
	}
	return dropped
}
