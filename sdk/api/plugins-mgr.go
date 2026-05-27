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
}
