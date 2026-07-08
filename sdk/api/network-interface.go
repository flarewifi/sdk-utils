/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "net"

// INetworkInterface represents a network interface in the system (/etc/config/network).
type INetworkInterface interface {

	// Returns the name of the interface.
	Ifname() string

	// Returns the device used for this interface.
	Device() (INetworkDevice, error)

	// Returns the status of the network interface.
	Up() bool

	// Returns the IPv4 address of the network interface.
	IpV4Addr() (*NetworkIpv4, error)

	// Returns the IPv6 address of the network interface.
	// Returns an error if no IPv6 address is assigned.
	IpV6Addr() (*NetworkIpv6, error)

	// Returns the ip net value of the network interface.
	IPNet() (*net.IPNet, error)
}
