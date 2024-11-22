/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkcfg

// AppCfg is the application configuration.
type AppCfg struct {
	// Examples: en, zh
	Lang string `json:"lang"`

	// Examples: USD, PH, CNY
	Currency string `json:"currency"`

	// Application secret key
	Secret string `json:"secret"`
}

// IAppCfgApi is used to read and write application configuration.
type IAppCfgApi interface {
	Get() (AppCfg, error)
	Save(AppCfg) error
}
