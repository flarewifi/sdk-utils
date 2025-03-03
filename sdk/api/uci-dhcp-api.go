/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

// DhcpCfg represents the DHCP configuration
type DhcpCfg struct {
	Section   string
	Ifname    string
	StartIp   string
	Limit     uint
	LeaseHour uint
}

type IDhcpApi interface {
	GetSection(ifname string) (section string, ok bool)
	GetConfig(section string) (dhcp *DhcpCfg, ok bool)
	SetConfig(ifname string, cfg *DhcpCfg) error
}
