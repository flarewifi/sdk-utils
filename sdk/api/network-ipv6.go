/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

// NetworkIpv6 holds the IPv6 address and prefix length for a network interface.
type NetworkIpv6 struct {
	Addr      string
	PrefixLen int
}
