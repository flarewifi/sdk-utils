/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type NetDevType string

const (
	NetDevBridge NetDevType = "bridge"
	NetDevEther  NetDevType = "ethernet"
	NetDevWLAN   NetDevType = "wlan"
	NetDevVLAN   NetDevType = "vlan"
)

// INetworkDevice represents a network device.
type INetworkDevice interface {

	// Returns the name of the network device.
	Name() string

	// Returns the type of the network device.
	Type() NetDevType

	// Returns the MAC address of the network device.
	MacAddr() string

	// Returns the administrative status of the network device.
	Up() bool

	// Returns true if the physical link is connected (cable plugged in, signal detected).
	// For wireless devices, this indicates association status.
	Carrier() bool

	// Returns the link speed of the network device in Mbps.
	// Returns 1000 Mbps as fallback if speed cannot be detected or parsed.
	SpeedMbps() int

	// Returns the duplex mode of the network device ("full", "half", or "unknown").
	Duplex() string

	// Returns the names of bridge member ports, if device is bridge interface.
	BridgeMembers() []string

	// Returns the current receive bytes of the network device.
	RxBytes() uint

	// Returns the current transmit bytes of the network device.
	TxBytes() uint

	// Returns the current download rate in bytes per second.
	// Calculated from the difference in RxBytes since last call.
	// Returns 0 on the first call (no previous reading available).
	RxRate() uint64

	// Returns the current upload rate in bytes per second.
	// Calculated from the difference in TxBytes since last call.
	// Returns 0 on the first call (no previous reading available).
	TxRate() uint64

	// Returns true if the network device is a bridge interface.
	IsBridge() bool

	// Returns true if the network device is a VLAN interface.
	IsVlan() bool

	// Returns true if the network device is an IFB (Intermediate Functional Block) interface,
	// identified by the "-ifb" name suffix used for traffic-shaping shadow devices.
	IsIfb() bool

	// Returns true if the network device is an Ethernet interface.
	IsEthernet() bool

	// Returns true if the network device is a wireless (WLAN) interface.
	IsWireless() bool
}
