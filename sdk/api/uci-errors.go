/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

var (
	ErrNoDhcp    = &errNoDhcp{}
	ErrNotBridge = &errNotBridge{}
	ErrNoVlanID  = &errNoVlanID{}
	ErrNotBrVlan = &errNotBrVlan{}
)

func ErrEmptyDev(ifname string) *errEmptyDev {
	return &errEmptyDev{ifname}
}

type errNoDhcp struct{}

func (err *errNoDhcp) Error() string {
	return "DHCP server is not running on network interface."
}

type errNotBridge struct{}

func (err *errNotBridge) Error() string {
	return "Not a bridge device"
}

type errEmptyDev struct {
	ifname string
}

func (err *errEmptyDev) Error() string {
	return "Can't get device for interface " + err.ifname
}

type errNoVlanID struct{}

func (err errNoVlanID) Error() string {
	return "No VLAN ID"
}

type errNotBrVlan struct{}

func (err errNotBrVlan) Error() string {
	return "Not a bridge-vlan device"
}

type errNoInterface struct{}

func (err errNoInterface) Error() string {
	return "Invalid network interface name"
}
