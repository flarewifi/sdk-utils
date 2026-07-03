package config

// interfacesConfigVersion is the on-disk schema version for interfaces.json.
// Bump it (and add a migration in ReadInterfacesConfig) only on a
// backward-incompatible change. Additive fields do NOT need a bump.
const interfacesConfigVersion = 1

const interfacesJsonFile = "interfaces.json"

// InterfaceRole classifies an interface. It is stored per-interface so the same
// file can describe LANs today and WAN uplinks when failover / load-balancing
// lands — no migration needed to introduce a WAN.
type InterfaceRole string

const (
	// RoleLan is a client-facing LAN. The empty role is also treated as LAN so
	// pre-existing configs (and interfaces with no explicit entry) keep working.
	RoleLan InterfaceRole = "lan"
	// RoleWan is an internet uplink, governed by WanCfg. Reserved for the future
	// WAN failover / balancing feature; unused by the captive-portal logic today.
	RoleWan InterfaceRole = "wan"
)

// WAN pool modes (reserved for the future WAN failover / balancing feature).
const (
	WanModeSingle   = ""         // one active WAN (default)
	WanModeFailover = "failover" // one active WAN, others stand by (by Priority)
	WanModeBalance  = "balance"  // spread traffic across WANs (by Weight)
)

// InterfacesCfg persists the admin's per-machine interface roles.
//
// LAN today:
//   - MainInterface: the ifname of the "main" LAN. Its IP address is where the
//     captive portal / custom domain is served, and it is the split-horizon DNS
//     target and the DNAT redirect target for every managed interface (so clients
//     on a different subnet, e.g. 20.0.0.0/20, reach 10.0.0.1).
//   - Interfaces[ifname].Managed: whether the app manages that LAN. Managed
//     interfaces get the captive portal + session firewall (custom nftables
//     rules). UNMANAGED interfaces are left completely untouched — no custom
//     firewall rules, their traffic flows as if the app were not installed.
//
// WAN later (fields already present, not yet consumed):
//   - WanPolicy.Mode + Interfaces[ifname].Wan describe a WAN uplink pool so the
//     failover / load-balancing feature can be added without changing this shape.
//
// This is intentionally separate from bandwidth.json (which owns per-interface
// traffic shaping): interfaces.json owns roles/portal, bandwidth.json owns caps.
type InterfacesCfg struct {
	Version       int                     `json:"version"`
	MainInterface string                  `json:"main_interface"`
	WanPolicy     WanPolicy               `json:"wan_policy"`
	Interfaces    map[string]InterfaceCfg `json:"interfaces"`
}

// WanPolicy is the pool-wide WAN behavior. Reserved for the future WAN feature.
type WanPolicy struct {
	Mode string `json:"mode"` // WanModeSingle | WanModeFailover | WanModeBalance
}

// InterfaceCfg is the per-interface entry. The relevant fields depend on Role:
// LAN uses Managed + CaptivePortal; WAN uses Wan. Keeping both on one struct
// (with Wan as an omitempty pointer) means a LAN entry stays compact while a WAN
// uplink is a drop-in addition.
type InterfaceCfg struct {
	Role InterfaceRole `json:"role"`
	// Managed: the app applies its custom firewall rules (session enforcement,
	// anti-tethering) to this interface.
	Managed bool `json:"managed"`
	// CaptivePortal: the port-80 captive-portal redirect applies. Only meaningful
	// when Managed is true — the redirect requires management.
	CaptivePortal bool    `json:"captive_portal"`
	Wan           *WanCfg `json:"wan,omitempty"`
}

// WanCfg holds the forward-looking WAN uplink settings for the failover /
// balancing feature. Defined now so interfaces.json is schema-compatible when
// that feature lands; nothing reads these fields yet.
type WanCfg struct {
	Enabled  bool `json:"enabled"`  // participates in the WAN pool
	Priority int  `json:"priority"` // failover order (lower = preferred)
	Weight   int  `json:"weight"`   // load-balance share (higher = more traffic)
	Metric   int  `json:"metric"`   // route metric applied to this uplink
}

func ReadInterfacesConfig() (InterfacesCfg, error) {
	var cfg InterfacesCfg
	err := readConfigFile(interfacesJsonFile, &cfg)
	if cfg.Interfaces == nil {
		cfg.Interfaces = map[string]InterfaceCfg{}
	}
	// Future backward-incompatible changes migrate here based on cfg.Version.
	return cfg, err
}

func WriteInterfacesConfig(cfg InterfacesCfg) error {
	if cfg.Interfaces == nil {
		cfg.Interfaces = map[string]InterfaceCfg{}
	}
	cfg.Version = interfacesConfigVersion
	return writeConfigFile(interfacesJsonFile, cfg)
}

// DefaultManagedDevice is the one L3 device that is managed out of the box: the
// primary LAN bridge. Every other interface is opt-in (unmanaged by default).
const DefaultManagedDevice = "br-lan"

// IsManaged reports whether the app manages a LAN (applies the captive portal +
// session firewall). An explicit admin choice always wins. With NO explicit
// entry the default is opt-in: only the primary LAN bridge (DefaultManagedDevice
// = "br-lan") is managed; any other newly-appeared interface is left untouched
// until an admin turns it on from the Interfaces page. A WAN-role interface is
// never a managed LAN. device is the interface's L3 device (e.g. "br-lan").
func (c InterfacesCfg) IsManaged(ifname, device string) bool {
	ic, ok := c.Interfaces[ifname]
	if !ok {
		return device == DefaultManagedDevice
	}
	if ic.Role == RoleWan {
		return false
	}
	return ic.Managed
}

// IsCaptive reports whether the captive-portal redirect applies to a LAN — i.e.
// the interface is BOTH managed AND has the captive portal enabled. With no
// explicit entry, the default-managed device (br-lan) also defaults to captive;
// every other interface is not captive by default. A non-managed interface is
// never captive.
func (c InterfacesCfg) IsCaptive(ifname, device string) bool {
	if !c.IsManaged(ifname, device) {
		return false
	}
	ic, ok := c.Interfaces[ifname]
	if !ok {
		return device == DefaultManagedDevice
	}
	return ic.CaptivePortal
}

// IsWan reports whether ifname is configured as a WAN uplink.
func (c InterfacesCfg) IsWan(ifname string) bool {
	ic, ok := c.Interfaces[ifname]
	return ok && ic.Role == RoleWan
}
