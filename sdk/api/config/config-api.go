/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkcfg

// IConfigApi is used to access the configuration API.
type IConfigApi interface {

	// Get the application configuration api.
	Application() IAppCfgApi

	// Get the bandwidth configuration api of a network interface.
	Bandwidth(ifname string) IBandwidthCfgApi
}
