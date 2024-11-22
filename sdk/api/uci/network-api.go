/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkuci

// INetworkApi is used to get/set network configuration
type INetworkApi interface {
	// device
	GetDevice(section string) (dev string, err error)
	GetDeviceSec(name string) (section string, err error)
	GetDeviceType(section string) (t string, err error)

	// bridge
	GetBridgeSecs() (sections []string)
	GetBridgeVlanFilter(section string) (enabled bool)
	GetBridgePorts(section string) (ports []string, err error)
	SetBridgePorts(section string, ports []string) error
	SetBridgeVlanFilter(section string, enabled bool) error

	// bridge-vlan
	CreateBrVlan(brvlan *BrVlan, ports []*BrVlanPort) error
	GetBrVlanSecs() (sections []string)
	GetBrVlanSec(brvlan *BrVlan) (section string, ok bool)
  GetBrVlanID(brvlan *BrVlan) (vlanid int, ok bool)
	GetBrVlanPorts(brvlan *BrVlan) (ports []*BrVlanPort, err error)
	SetBrVlanPorts(brvlan *BrVlan, ports []*BrVlanPort) error
	DeleteBrVlan(brvlan *BrVlan)

	// interface
	GetInterface(section string) (iface *INetIface, err error)
	GetInterfaceSecs() (sections []string)
  GetInterfaces() (ifaces []*INetIface, err error)
	SetInterface(section string, cfg *INetIface) error
}
