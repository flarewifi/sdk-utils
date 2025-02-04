/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "os"

type IPluginCfgApi interface {
	// Write a value to the plugin configuration file.
	Write(path string, value []byte) error

	// Read a value from the plugin configuration file.
	Read(path string) ([]byte, error)

	// List entries inside the path
	List(path string) ([]*ConfigEntry, error)
}

type ConfigEntry struct {
	Entry os.DirEntry
	Path  string
}
