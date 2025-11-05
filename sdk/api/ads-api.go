/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

// IAdsApi is used for displaying ads in the captive portal.
type IAdsApi interface {
	// Init initializes the ads API with the given app ID.
	Init(appId string)
}
