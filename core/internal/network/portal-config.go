package network

import (
	"sort"
	"sync"

	"core/internal/modules/captivedns"
	"core/internal/modules/nftables"
	"core/utils/config"
)

// portalCfgMu serializes the read-resolve-apply sequence in ApplyPortalConfig so
// a boot, an interface-up event, and an admin save can't interleave. The
// underlying nftables ops are already serialized internally; this guards the
// higher-level "resolve main IP then push every interface's mode" transaction.
var portalCfgMu sync.Mutex

// LanInfo is a read-only snapshot of a registered LAN for the admin Interfaces
// page. It combines the cached registry data (name, IPv4, CIDR) with live
// device/up state from UBUS.
type LanInfo struct {
	Name    string // UBUS interface name, e.g. "lan"
	Device  string // L3 device, e.g. "br-lan"
	IPv4    string // e.g. "10.0.0.1"
	Netmask int    // prefix length, e.g. 20
	CIDR    string // subnet, e.g. "10.0.0.0/20"
	Up      bool
}

// ListLanInfos returns a snapshot of every registered LAN (the interfaces the
// admin can manage on the Interfaces page), sorted by name for a stable UI.
func ListLanInfos() []LanInfo {
	registry.mu.RLock()
	names := make([]string, 0, registry.count)
	base := make(map[string]LanInfo, registry.count)
	for _, e := range registry.byIp {
		names = append(names, e.name)
		base[e.name] = LanInfo{Name: e.name, IPv4: e.ipv4Addr, CIDR: e.cidrString}
	}
	registry.mu.RUnlock()

	infos := make([]LanInfo, 0, len(names))
	for _, n := range names {
		li := base[n]
		iface := NewNetworkInterface(n)
		if info, err := iface.getInfo(); err == nil {
			li.Device = info.Device
			li.Up = info.Up
		}
		if ip, err := iface.IpV4Addr(); err == nil {
			li.Netmask = ip.Netmask
			if li.IPv4 == "" {
				li.IPv4 = ip.Addr
			}
		}
		infos = append(infos, li)
	}

	sort.Slice(infos, func(i, j int) bool { return infos[i].Name < infos[j].Name })
	return infos
}

// MainInterface returns the resolved main LAN ifname (the configured one when it
// is still a registered LAN, else the first LAN by name), and "" if there are no
// LANs. This is the interface whose IP hosts the portal / custom domain.
func MainInterface() string {
	cfg, _ := config.ReadInterfacesConfig()
	return resolveMainInterface(cfg, lanNames())
}

// IsClientIPManaged reports whether the LAN interface whose subnet contains
// clientIP is managed by the app. known is false when clientIP maps to no
// registered LAN (an unidentifiable client); callers should treat that as
// "don't act" rather than assuming unmanaged.
func IsClientIPManaged(clientIP string) (managed, known bool) {
	lan, err := FindByIp(clientIP)
	if err != nil {
		// clientIP is in no registered LAN subnet — an unidentifiable client
		// (a non-LAN interface such as tailscale0 / a VPN, or the machine/proxy
		// itself). Not known, so callers treat it as "not managed".
		return false, false
	}

	// The client is on a registered LAN, so it is a KNOWN interface regardless of
	// what follows. getInfo only resolves the L3 device name used for the
	// default-managed decision (br-lan defaults managed); a transient UBUS failure
	// must NOT downgrade a real LAN client to "unknown" and bounce it to /admin.
	// An explicit interfaces.json entry is keyed by name and needs no device name.
	device := ""
	if info, ierr := NewNetworkInterface(lan.Name()).getInfo(); ierr == nil {
		device = info.Device
	}
	cfg, _ := config.ReadInterfacesConfig()
	return cfg.IsManaged(lan.Name(), device), true
}

// ApplyPortalConfig reconciles the live firewall + DNS with interfaces.json:
//   - every registered LAN is marked managed or unmanaged (SetInterfaceMode);
//     unmanaged interfaces get no custom rules and flow through untouched,
//   - the shared port-80 DNAT target is set to the MAIN LAN's IP so managed
//     clients on any subnet reach the main portal (SetCaptivePortalTarget),
//   - split-horizon DNS + the RFC 8910 advertisement point at that same main IP.
//
// It is safe to call repeatedly (idempotent) — at boot, on an interface-up
// event, and after an admin save — so changes apply immediately. A missing
// interfaces.json yields defaults (every LAN captive, main = first LAN), exactly
// reproducing the pre-feature behavior.
func ApplyPortalConfig() error {
	portalCfgMu.Lock()
	defer portalCfgMu.Unlock()

	cfg, _ := config.ReadInterfacesConfig()
	lans := FindAll()

	names := make([]string, 0, len(lans))
	for _, l := range lans {
		names = append(names, l.Name())
	}

	mainIf := resolveMainInterface(cfg, names)

	// Resolve the main LAN's IPv4/IPv6 — the portal DNAT target and DNS answer.
	var mainIp4, mainIp6 string
	if mainIf != "" {
		iface := NewNetworkInterface(mainIf)
		if ip, err := iface.IpV4Addr(); err == nil {
			mainIp4 = ip.Addr
		}
		if ip, err := iface.IpV6Addr(); err == nil {
			mainIp6 = ip.Addr
		}
	}

	// Mark each LAN device managed/unmanaged and captive/not.
	for _, l := range lans {
		info, err := NewNetworkInterface(l.Name()).getInfo()
		if err != nil || info.Device == "" {
			continue
		}
		managed := cfg.IsManaged(l.Name(), info.Device)
		captive := cfg.IsCaptive(l.Name(), info.Device)
		if err := nftables.SetInterfaceMode(info.Device, managed, captive); err != nil {
			return err
		}
	}

	// One shared DNAT rule redirects every captive interface's port-80 traffic to
	// the main LAN IP (e.g. 10.0.0.1), regardless of the client's subnet.
	if err := nftables.SetCaptivePortalTarget(mainIp4, mainIp6); err != nil {
		return err
	}

	// Point the portal hostname at the main LAN IP (no-op in dev/devkit where
	// there is no portal domain). Best-effort: a DNS failure must not abort the
	// firewall reconcile above.
	_ = captivedns.Setup(mainIp4)

	return nil
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// lanNames returns the names of all registered LANs.
func lanNames() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	names := make([]string, 0, registry.count)
	for _, e := range registry.byIp {
		names = append(names, e.name)
	}
	return names
}

// resolveMainInterface picks the effective main LAN: the configured one if it is
// still a registered LAN, otherwise the first LAN by name (deterministic), or ""
// when there are no LANs.
func resolveMainInterface(cfg config.InterfacesCfg, lanNames []string) string {
	if cfg.MainInterface != "" {
		for _, n := range lanNames {
			if n == cfg.MainInterface {
				return cfg.MainInterface
			}
		}
	}
	if len(lanNames) == 0 {
		return ""
	}
	sorted := append([]string(nil), lanNames...)
	sort.Strings(sorted)
	return sorted[0]
}
