/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

// AppConfig is the application configuration.
type AppConfig struct {
	// Examples: en, zh
	Lang string `json:"lang"`

	// Examples: USD, PH, CNY
	Currency string `json:"currency"`

	// Application secret key
	Secret string `json:"secret"`

	// Application channel: development, beta, stable
	Channel string `json:"channel"`
}

// IAppCfgApi is used to read and write application configuration.
type IAppCfgApi interface {
	Get() (AppConfig, error)
	Save(AppConfig) error
}
