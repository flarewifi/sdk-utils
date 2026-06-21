/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// PluginsConfigPath returns the absolute path to data/config/plugins.json.
func PluginsConfigPath() string {
	return filepath.Join(sdkutils.PathConfigDir, "plugins.json")
}

// ReadPluginsConfig reads and parses data/config/plugins.json into the shared
// sdkutils.PluginsConfig shape (installed plugin metadata + meta-plugin bundle
// records).
func ReadPluginsConfig() (sdkutils.PluginsConfig, error) {
	var cfg sdkutils.PluginsConfig
	if err := sdkutils.JsonRead(PluginsConfigPath(), &cfg); err != nil {
		return sdkutils.PluginsConfig{}, err
	}
	return cfg, nil
}

// WritePluginsConfig writes cfg back to data/config/plugins.json. It round-trips
// the whole struct, so callers should mutate a value obtained from
// ReadPluginsConfig rather than constructing a partial one.
func WritePluginsConfig(cfg sdkutils.PluginsConfig) error {
	return sdkutils.JsonWrite(PluginsConfigPath(), cfg)
}
