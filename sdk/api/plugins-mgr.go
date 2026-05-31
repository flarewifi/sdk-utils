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

	// InstallPlugin installs a plugin from any source and registers it live
	// without requiring a server restart. The source is selected by def.Src:
	//   - store:            resolves the release download URL internally via the
	//     core RPC service (from def.StorePackage / def.StorePluginVersion) and
	//     installs from an encrypted scratch disk. The resolved URL is transient
	//     and is never persisted as part of the source definition.
	//   - git:              clones def.GitURL at def.GitRef (a branch, tag, or
	//     commit hash) into a temp directory.
	//   - local/system/zip: installs from def.LocalPath, a source folder already
	//     on disk (it must contain a plugin.json); the source files are left in
	//     place.
	InstallPlugin(def sdkutils.PluginSrcDef) error

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
}
