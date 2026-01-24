/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type PaymentOption struct {
	UUID        string // Unique, stable identifier (16-char hash based on device property like MAC address)
	Name        string // Display label for the user
	RouteName   string
	RouteParams map[string]string
}
