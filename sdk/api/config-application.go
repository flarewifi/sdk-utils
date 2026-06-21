/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import sdkutils "github.com/flarewifi/sdk-utils"

type SupportedLanguage struct {
	Code string
	Name string
}

// AppConfig is the application configuration.
type AppConfig sdkutils.AppConfig

// IAppCfgApi is used to read and write application configuration.
type IAppCfgApi interface {
	Get() (AppConfig, error)
	Save(AppConfig) error
}
