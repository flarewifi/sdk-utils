/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

// LanInterfaceCfg is the persisted admin configuration for a single LAN interface.
type LanInterfaceCfg struct {
	// EnableCaptivePortal is the single authority for whether this interface gets
	// traffic shaping and the captive session firewall rules.
	EnableCaptivePortal bool

	// IpAddress is the desired static IP for the interface. Only takes effect
	// once the machine applies it to the OS network config.
	IpAddress string

	// Netmask is the desired static netmask for the interface.
	Netmask string
}

// InterfaceCfg is the full persisted network interface configuration.
type InterfaceCfg struct {
	// PortalInterface is the name of the interface designated as the main
	// captive-portal interface. If set, it must reference a captive-enabled
	// entry in LanInterfaces.
	PortalInterface string

	// LanInterfaces maps an interface name to its configuration.
	LanInterfaces map[string]LanInterfaceCfg
}

// IInterfaceCfgApi is used to get and set network interface configuration
// (captive portal enablement, static IP/netmask, and the main portal interface).
type IInterfaceCfgApi interface {
	// Get returns the current interface configuration.
	Get() (InterfaceCfg, error)

	// Save persists the interface configuration and applies it to the running
	// system (nftables, DNS, traffic control). Returns an error if
	// PortalInterface is set but does not reference a captive-enabled
	// interface in LanInterfaces.
	Save(cfg InterfaceCfg) error
}
