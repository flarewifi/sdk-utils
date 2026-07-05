package network

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"core/internal/modules/captivedns"
	"core/internal/modules/nftables"
	"core/internal/modules/ubus"
	"core/utils/config"
)

// portalCfgMu serializes the read-resolve-apply sequence in ApplyPortalConfig so
// a boot, an interface-up event, and an admin save can't interleave. The
// underlying nftables ops are already serialized internally; this guards the
// higher-level "resolve main IP then push every interface's mode" transaction.
var portalCfgMu sync.Mutex

// LanInfo is a read-only snapshot of a LAN candidate for the admin Interfaces
// page. It combines the UBUS interface name with live device/IP/up state.
type LanInfo struct {
	Name    string // UBUS interface name, e.g. "lan"
	Device  string // L3 device, e.g. "br-lan"
	IPv4    string // e.g. "10.0.0.1"
	Netmask int    // prefix length, e.g. 20
	CIDR    string // subnet, e.g. "10.0.0.0/20"
	Up      bool
}

// ListLanInfos returns every LAN candidate the admin can manage on the Interfaces
// page — enumerated live from UBUS, NOT from the registry, so an interface that
// is not yet captive (and therefore unregistered) still appears and can be turned
// on. WAN uplinks and loopback are excluded. Sorted by name for a stable UI.
func ListLanInfos() []LanInfo {
	names, err := ubus.GetInterfaceNames()
	if err != nil {
		return nil
	}

	infos := make([]LanInfo, 0, len(names))
	for _, n := range names {
		if isNonLanInterface(n) {
			continue
		}
		li := LanInfo{Name: n}
		iface := NewNetworkInterface(n)
		if info, err := iface.getInfo(); err == nil {
			li.Device = info.Device
			li.Up = info.Up
		}
		if ip, err := iface.IpV4Addr(); err == nil {
			li.IPv4 = ip.Addr
			li.Netmask = ip.Netmask
			li.CIDR = fmt.Sprintf("%s/%d", ip.Addr, ip.Netmask)
		}
		infos = append(infos, li)
	}

	sort.Slice(infos, func(i, j int) bool { return infos[i].Name < infos[j].Name })
	return infos
}

// isNonLanInterface excludes interfaces that are never client-facing LANs from
// the Interfaces page: the loopback and any WAN uplink. Everything else is a LAN
// candidate the admin may enable the captive portal on.
func isNonLanInterface(name string) bool {
	switch name {
	case "loopback", "lo":
		return true
	}
	return strings.HasPrefix(name, "wan")
}

// MainInterface returns the resolved main LAN ifname (the configured one when it
// is still a registered LAN, else the first LAN by name), and "" if there are no
// LANs. This is the interface whose IP hosts the portal / custom domain.
func MainInterface() string {
	cfg, _ := config.ReadInterfacesConfig()
	return resolveMainInterface(cfg, lanNameDevices())
}

// IsClientIPManaged reports whether clientIP belongs to a LAN interface that has
// the captive portal enabled — the ONLY traffic that should be funneled to the
// portal. It returns false when the IP maps to no registered LAN subnet (a
// non-LAN interface such as tailscale0 / a VPN, or the machine/proxy itself) or
// to a LAN whose captive portal is off; those clients are sent to the admin
// dashboard instead.
//
// The captive decision is config.IsCaptivePortalEnabled — the SAME authority the
// firewall/TC side uses (an explicit interfaces.json entry keyed by the
// interface NAME wins; with no entry the interface whose L3 DEVICE is the
// primary bridge, br-lan, is captive by default). Passing the device is what
// keeps the two sides consistent on a machine running on defaults: an
// interface-name heuristic here previously disagreed with the device-keyed
// default and bounced legitimate portal clients to /admin. The device comes
// from the LAN registry cache (resolved at registration, refreshed by
// updateLanCidr) — deliberately NOT a per-request UBUS lookup, so a transient
// UBUS failure can never misroute a client.
func IsClientIPManaged(clientIP string) bool {
	lan, err := FindByIp(clientIP)
	if err != nil {
		return false
	}

	cfg, _ := config.ReadInterfacesConfig()
	return cfg.IsCaptivePortalEnabled(lan.Name(), lanDevice(lan.Name()))
}

// ClassifyClientIP explains, as a one-line string, how IsClientIPManaged
// decides for clientIP — which LAN (if any) the IP matched, the LAN's cached
// device, and the resulting captive verdict. Diagnostic only: used by the
// middlewares to log WHY a request was classified unmanaged, since a
// misclassification silently bounces portal clients to /admin and is otherwise
// invisible on a production machine.
func ClassifyClientIP(clientIP string) string {
	lan, err := FindByIp(clientIP)
	if err != nil {
		return fmt.Sprintf("no registered LAN subnet matches (%v); registered: %s", err, describeRegistry())
	}

	cfg, _ := config.ReadInterfacesConfig()
	device := lanDevice(lan.Name())
	return fmt.Sprintf("lan=%s device=%q captive=%t", lan.Name(), device, cfg.IsCaptivePortalEnabled(lan.Name(), device))
}

// describeRegistry summarizes every registered LAN's name, cached device and
// CIDRs for the ClassifyClientIP no-match diagnostic.
func describeRegistry() string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	parts := make([]string, 0, registry.count)
	for _, e := range registry.byIp {
		parts = append(parts, fmt.Sprintf("%s(device=%q cidr=%q cidr6set=%t)", e.name, e.device, e.cidrString, e.cidr6 != nil))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// ApplyPortalConfig reconciles the live firewall + DNS with interfaces.json:
//   - every registered LAN is marked managed or unmanaged (SetInterfaceMode);
//     unmanaged interfaces get no custom rules and flow through untouched,
//   - the shared port-80 DNAT target is set to the MAIN LAN's IP so managed
//     clients on any subnet reach the main portal (SetCaptivePortalTarget),
//   - the portal-serving IPs (every captive LAN's gateway + the main IP) are
//     pushed to the portal_ips bypass sets so captive clients reach the portal
//     directly — never DNAT'd away from it, never dropped by the session gate
//     (SetPortalIPs),
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

	// Re-sync every LAN's cached device + CIDR with its live state BEFORE
	// resolving the main interface below. FindByIp / lanDevice (and so
	// IsClientIPManaged, which decides portal vs /admin) match clients against
	// this cache, and this reconcile is the event-independent convergence point:
	// an interface that had no IP at boot, or whose IP an admin just applied,
	// must become matchable here even if no ubus ifup event was (or ever is)
	// observed. Best-effort — a LAN that is down / still address-less simply
	// stays unmatchable.
	for _, l := range lans {
		_, _ = updateLanCidr(l.Name(), l)
	}

	mainIf := resolveMainInterface(cfg, lanNameDevices())

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

	// Mark each LAN device captive-or-free. EnableCaptivePortal is the single
	// authority: a captive interface gets both the session firewall and the
	// port-80 redirect (managed == captive), everything else is left untouched.
	// Along the way, collect every captive LAN's gateway IP: together with the
	// main IP these are the portal-serving addresses that clients on ANY captive
	// interface must be able to reach, so they feed the portal_ips bypass sets
	// (skip the port-80 DNAT + the forward-chain session gate).
	portalIPs := make([]string, 0, len(lans)*2+2)
	seenPortalIP := make(map[string]bool)
	addPortalIP := func(ip string) {
		if ip != "" && !seenPortalIP[ip] {
			seenPortalIP[ip] = true
			portalIPs = append(portalIPs, ip)
		}
	}
	addPortalIP(mainIp4)
	addPortalIP(mainIp6)

	captiveIfnames := make([]string, 0, len(lans))
	freeIfnames := make([]string, 0, len(lans))

	for _, l := range lans {
		iface := NewNetworkInterface(l.Name())
		info, err := iface.getInfo()
		if err != nil || info.Device == "" {
			continue
		}
		enable := cfg.IsCaptivePortalEnabled(l.Name(), info.Device)
		if err := nftables.SetInterfaceMode(info.Device, enable, enable); err != nil {
			return err
		}
		if enable {
			captiveIfnames = append(captiveIfnames, l.Name())
			if ip, err := iface.IpV4Addr(); err == nil {
				addPortalIP(ip.Addr)
			}
			if ip, err := iface.IpV6Addr(); err == nil {
				addPortalIP(ip.Addr)
			}
		} else {
			freeIfnames = append(freeIfnames, l.Name())
		}
	}

	// Portal-reachability bypass: push the portal-serving IPs so captive clients
	// can reach the portal on any of them without being DNAT'd or session-gated.
	if err := nftables.SetPortalIPs(portalIPs); err != nil {
		return err
	}

	// One shared DNAT rule redirects every captive interface's port-80 traffic to
	// the main LAN IP (e.g. 10.0.0.1), regardless of the client's subnet.
	if err := nftables.SetCaptivePortalTarget(mainIp4, mainIp6); err != nil {
		return err
	}

	// Point the portal hostname at the main LAN IP and advertise the RFC 8910
	// portal URL (DHCP option 114) on every captive interface's DHCP pool —
	// removing it from pools that are no longer captive (no-op in dev/devkit
	// where there is no portal domain). Best-effort: a DNS failure must not
	// abort the firewall reconcile above.
	_ = captivedns.Setup(mainIp4, captiveIfnames, freeIfnames)

	return nil
}

// ReconcileInterfaces brings live traffic control in line with the current
// interfaces.json captive set, then re-applies the portal firewall + DNS. A newly
// captive interface has its TC set up; an interface no longer captive has it torn
// down. Registration + event listeners are static (established at boot for every
// LAN candidate), so this only toggles per-interface shaping — no goroutine or
// registry churn. Safe to call repeatedly (idempotent), so an admin toggling the
// captive portal on the Interfaces page applies live without a restart.
func ReconcileInterfaces() error {
	cfg, _ := config.ReadInterfacesConfig()

	for _, lan := range FindAll() {
		device := ""
		if info, err := NewNetworkInterface(lan.Name()).getInfo(); err == nil {
			device = info.Device
		}
		captive := cfg.IsCaptivePortalEnabled(lan.Name(), device)

		switch {
		case captive && !lan.HasTrafficControl():
			if err := lan.SetupTrafficControl(); err != nil {
				return err
			}
		case !captive && lan.HasTrafficControl():
			lan.TeardownTrafficControl()
		}
	}

	return ApplyPortalConfig()
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// IsCaptivePortalEnabled reports whether the LAN ifname currently has the captive
// portal enabled (and therefore gets traffic shaping + the session firewall),
// resolving its L3 device so the primary-LAN default is honored. Exported for the
// SDK so plugins can scope per-interface work (e.g. bandwidth) to captive LANs.
func IsCaptivePortalEnabled(ifname string) bool {
	cfg, _ := config.ReadInterfacesConfig()
	device := ""
	if info, err := NewNetworkInterface(ifname).getInfo(); err == nil {
		device = info.Device
	}
	if device == "" {
		// Transient UBUS failure — fall back to the registry's cached device so
		// the device-keyed captive default (br-lan) is still honored rather than
		// silently deciding "not captive".
		device = lanDevice(ifname)
	}
	return cfg.IsCaptivePortalEnabled(ifname, device)
}

// lanIsCaptive is the internal alias used by the network package.
func lanIsCaptive(ifname string) bool { return IsCaptivePortalEnabled(ifname) }

// lanNameDevice pairs a registered LAN's interface name with its cached L3
// device — the two keys the captive decision needs (explicit config entries are
// keyed by NAME; the out-of-the-box default is keyed by DEVICE, br-lan).
type lanNameDevice struct {
	name   string
	device string
}

// lanNameDevices returns every registered LAN's (name, device) pair, sorted by
// name for deterministic resolution. Reads only the registry cache — no UBUS.
func lanNameDevices() []lanNameDevice {
	registry.mu.RLock()
	pairs := make([]lanNameDevice, 0, registry.count)
	for _, e := range registry.byIp {
		pairs = append(pairs, lanNameDevice{name: e.name, device: e.device})
	}
	registry.mu.RUnlock()

	sort.Slice(pairs, func(i, j int) bool { return pairs[i].name < pairs[j].name })
	return pairs
}

// resolveMainInterface picks the effective main LAN — the interface whose IP
// hosts the portal (DNAT target + split-horizon DNS answer):
//  1. the configured portal_interface, if it is still a registered LAN;
//  2. else the first (by name) CAPTIVE LAN — with no config this is the LAN on
//     the primary bridge (device br-lan), matching the firewall's default, so a
//     fresh machine never points DNS/DNAT at a non-captive interface that
//     merely sorts first;
//  3. else the first LAN by name (deterministic), or "" with no LANs at all.
func resolveMainInterface(cfg config.InterfacesCfg, lans []lanNameDevice) string {
	if cfg.PortalInterface != "" {
		for _, l := range lans {
			if l.name == cfg.PortalInterface {
				return cfg.PortalInterface
			}
		}
	}
	if len(lans) == 0 {
		return ""
	}
	for _, l := range lans {
		if cfg.IsCaptivePortalEnabled(l.name, l.device) {
			return l.name
		}
	}
	return lans[0].name
}
