package config

// interfacesConfigVersion is the on-disk schema version for interfaces.json.
// Bump it (and add a migration in ReadInterfacesConfig) only on a
// backward-incompatible change. Additive fields do NOT need a bump.
const interfacesConfigVersion = 1

const interfacesJsonFile = "interfaces.json"

// DefaultPrimaryLan is the one L3 device that is captive out of the box: the
// primary LAN bridge. With no explicit config entry this device defaults to
// captive-portal enabled (so a fresh machine serves the portal + traffic shaping
// without manual setup); every other interface is opt-in (left free by default).
const DefaultPrimaryLan = "br-lan"

// InterfacesCfg persists the admin's per-machine LAN interface settings.
//
//   - PortalInterface: the ifname whose IP hosts the captive portal / custom
//     domain. It is the split-horizon DNS target and the shared port-80 DNAT
//     target for every captive interface, so clients on any captive subnet reach
//     that one address.
//   - LanInterfaces[ifname].EnableCaptivePortal: the SINGLE authority. When set,
//     the interface is the only kind that gets BOTH traffic shaping (tc) AND the
//     custom firewall rules (session enforcement + captive redirect). Every other
//     interface/device is left completely free to be configured any way.
//   - LanInterfaces[ifname].IpAddress / Netmask: the desired static IP for the
//     interface. Stored on save; pushed to the OS only by the explicit "Apply
//     Changes" action (UCI write + netifd reload).
type InterfacesCfg struct {
	Version         int                        `json:"version"`
	PortalInterface string                     `json:"portal_interface"`
	LanInterfaces   map[string]LanInterfaceCfg `json:"lan_interfaces"`
}

type LanInterfaceCfg struct {
	EnableCaptivePortal bool   `json:"enable_captive_portal"`
	IpAddress           string `json:"ip_address"`
	Netmask             string `json:"netmask"`
}

func ReadInterfacesConfig() (InterfacesCfg, error) {
	var cfg InterfacesCfg
	err := readConfigFile(interfacesJsonFile, &cfg)
	if cfg.LanInterfaces == nil {
		cfg.LanInterfaces = map[string]LanInterfaceCfg{}
	}
	// Future backward-incompatible changes migrate here based on cfg.Version.
	return cfg, err
}

func WriteInterfacesConfig(cfg InterfacesCfg) error {
	if cfg.LanInterfaces == nil {
		cfg.LanInterfaces = map[string]LanInterfaceCfg{}
	}
	cfg.Version = interfacesConfigVersion
	return writeConfigFile(interfacesJsonFile, cfg)
}

// IsCaptivePortalEnabled reports whether the captive portal (and therefore tc
// shaping + the session firewall) is enabled for a LAN. An explicit config entry
// always wins. With NO entry the default is opt-in: only the primary LAN bridge
// (DefaultPrimaryLan = "br-lan") is captive, so a fresh machine works out of the
// box; every other interface is left free until an admin enables it. device is
// the interface's L3 device (e.g. "br-lan").
func (c InterfacesCfg) IsCaptivePortalEnabled(ifname, device string) bool {
	ic, ok := c.LanInterfaces[ifname]
	if !ok {
		return device == DefaultPrimaryLan
	}
	return ic.EnableCaptivePortal
}
