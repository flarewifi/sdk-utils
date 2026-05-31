/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import sdkutils "github.com/flarehotspot/sdk-utils"

// IPluginsMgrApi is used to get data of installed plugins in the system.
type IPluginsMgrApi interface {

	// Find a plugin by name as defined in package.yml "name" field.
	FindByName(name string) (IPluginApi, bool)

	// Find a plugin by path as defined in package.yml "package" field.
	FindByPkg(pkg string) (IPluginApi, bool)

	// Returns all plugins installed in the system.
	All() []IPluginApi

	// InstallFromStore downloads a plugin zip from the marketplace, installs it
	// into plugins/installed using an encrypted scratch disk, and registers it
	// live without requiring a server restart. The zipURL is used only for the
	// download and is never persisted as part of the plugin's source definition.
	InstallFromStore(def sdkutils.PluginSrcDef, zipURL string) error

	// Uninstall marks a plugin for removal on the next restart.
	Uninstall(pkg string) error

	// IsToBeRemoved returns true if the plugin has been marked for removal.
	IsToBeRemoved(pkg string) bool

	// HasPendingUpdate returns true if a downloaded update is waiting to be applied.
	HasPendingUpdate(pkg string) bool

	// SourceDef returns the source definition for an installed plugin —
	// where it came from and how it was installed (git / store / system /
	// local / zip). Returns (zero-value, false) if the package is not
	// installed.
	SourceDef(pkg string) (sdkutils.PluginSrcDef, bool)

	// InstallMetaMember installs a plugin as a member of the named meta plugin
	// and registers it live. The plugin's metadata records metaPkg as an owner
	// rather than marking it a standalone (user-initiated) install.
	InstallMetaMember(def sdkutils.PluginSrcDef, zipURL string, metaPkg string) error

	// AddMetaOwner records that metaPkg owns an already-installed member without
	// reinstalling it.
	AddMetaOwner(memberPkg string, metaPkg string) error

	// RemoveMetaOwner drops metaPkg from a member's owners. Used to roll back a
	// partially-completed meta install for members that were only adopted (not
	// freshly installed).
	RemoveMetaOwner(memberPkg string, metaPkg string) error

	// SaveMetaRecord persists an installed meta plugin's record.
	SaveMetaRecord(rec sdkutils.MetaInstallRecord) error

	// MetaRecord returns the install record for a meta plugin and whether it
	// exists.
	MetaRecord(pkg string) (sdkutils.MetaInstallRecord, bool)

	// MetaRecords returns all installed meta plugin records.
	MetaRecords() []sdkutils.MetaInstallRecord

	// UninstallMeta removes a meta plugin: it drops the meta from each member's
	// owners and marks for removal any member left with no owners that was not
	// installed standalone. Member removals apply on the next restart.
	UninstallMeta(pkg string) error

	// MetaMembership reports whether a plugin was installed standalone and which
	// meta plugins own it as a member. ok is false when the package has no
	// recorded metadata.
	MetaMembership(pkg string) (standalone bool, owners []string, ok bool)
}
