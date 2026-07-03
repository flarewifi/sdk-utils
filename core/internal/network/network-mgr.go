package network

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"core/internal/modules/ubus"
	"core/utils/config"
	jobque "core/utils/job-que"
)

const defaultSpeed int = DefaultLinkSpeed // fallback speed in Mbps when link speed cannot be detected

// =============================================================================
// LAN REGISTRY — replaces sync.Map for O(1) count, fast reads, cached CIDRs
// =============================================================================

// lanEntry caches all data needed for IP matching so FindByIp() never
// calls UBUS or parses CIDR strings at lookup time.
type lanEntry struct {
	name       string
	lan        *NetworkLan
	device     string     // cached L3 device, e.g. "br-lan" — may be "" if never resolved
	cidr       *net.IPNet // pre-parsed IPv4 CIDR — may be nil if no IPv4 address
	cidr6      *net.IPNet // pre-parsed IPv6 CIDR — may be nil if no IPv6 address
	ipv4Addr   string     // e.g. "192.168.1.1"
	ipv6Addr   string     // e.g. "2001:db8::1"
	cidrString string     // e.g. "192.168.1.0/24"
}

// lanRegistry is the optimized replacement for sync.Map.
// It uses a RWMutex instead of sync.Map because reads vastly outnumber writes
// (LANs are added once at startup, then only read).
type lanRegistry struct {
	mu     sync.RWMutex
	byName map[string]*lanEntry // O(1) name lookup
	byIp   []*lanEntry          // ordered slice for linear scan (fast for 1-10 LANs)
	count  int                  // cached length — O(1) GetLanCount()
}

var registry = &lanRegistry{
	byName: make(map[string]*lanEntry),
	byIp:   make([]*lanEntry, 0, 4), // pre-allocate for typical 1-3 LAN setups
}

var netQueue = jobque.NewJobQueue[any]()

// =============================================================================
// INTERNAL HELPERS
// =============================================================================

// addLan builds a registry entry for the LAN (caching its parsed CIDR) and stores
// it. IPv4 is optional at registration time: every LAN candidate is registered so
// it can be identified and toggled captive live, and a down / not-yet-configured
// interface may have no address yet — listenLanEvents fills the CIDR on the next
// ifup. Idempotent: re-registering an existing LAN replaces its entry in place.
func addLan(lan *NetworkLan) error {
	iface := lan.GetInterface()

	entry := &lanEntry{
		name: lan.Name(),
		lan:  lan,
	}

	// Cache the L3 device (e.g. "br-lan"). The out-of-the-box captive default in
	// config.IsCaptivePortalEnabled is keyed on the DEVICE, not the interface
	// name, and per-request consumers (IsClientIPManaged → portal-vs-admin) must
	// not pay a UBUS lookup — so the device is resolved here and refreshed by
	// updateLanCidr.
	if info, err := iface.getInfo(); err == nil {
		entry.device = info.Device
	}

	if ipv4, err := iface.IpV4Addr(); err == nil {
		cidrStr := fmt.Sprintf("%s/%d", ipv4.Addr, ipv4.Netmask)
		if _, cidr, err := net.ParseCIDR(cidrStr); err == nil {
			entry.cidr = cidr
			entry.ipv4Addr = ipv4.Addr
			entry.cidrString = cidrStr
		}
	}

	// IPv6 is optional — don't fail if absent
	if ipv6, err := iface.IpV6Addr(); err == nil {
		cidr6Str := fmt.Sprintf("%s/%d", ipv6.Addr, ipv6.PrefixLen)
		if _, cidr6, err := net.ParseCIDR(cidr6Str); err == nil {
			entry.cidr6 = cidr6
			entry.ipv6Addr = ipv6.Addr
		}
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	if _, exists := registry.byName[lan.Name()]; exists {
		for i, e := range registry.byIp {
			if e.name == lan.Name() {
				registry.byIp[i] = entry
				break
			}
		}
	} else {
		registry.byIp = append(registry.byIp, entry)
		registry.count++
	}
	registry.byName[lan.Name()] = entry

	return nil
}

// updateLanCidr rebuilds the cached IPv4 (and optionally IPv6) CIDR for a
// LAN whose IP may have changed (e.g. after an interface reinit or an admin
// "Apply Changes"). Reports whether the cached IPv4 CIDR actually changed so
// callers can skip a portal reconcile when nothing moved. Called from
// listenLanEvents on IfEventUp and from ApplyPortalConfig.
//
// The cache MUST track the live address: FindByIp — and therefore
// IsClientIPManaged, which gates the portal-vs-admin funnel — matches clients
// against these CIDRs. A stale/missing entry misclassifies every client on the
// interface as unmanaged and bounces them to /admin instead of the portal.
func updateLanCidr(lanName string, lan *NetworkLan) (changed bool, err error) {
	// One UBUS read covers everything the registry caches: device, IPv4, IPv6.
	info, err := lan.GetInterface().getInfo()
	if err != nil {
		return false, fmt.Errorf("failed to get interface info for LAN '%s': %w", lanName, err)
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	entry, ok := registry.byName[lanName]
	if !ok {
		return false, fmt.Errorf("LAN '%s' not found in registry", lanName)
	}

	// Refresh the cached L3 device first — even when the interface has no IP yet,
	// the device drives the captive default (device == br-lan) that
	// IsClientIPManaged relies on.
	if info.Device != "" && entry.device != info.Device {
		entry.device = info.Device
		changed = true
	}

	if len(info.IpV4Addresses) == 0 {
		return changed, fmt.Errorf("failed to get IPv4 for LAN '%s': no IPv4 addresses found", lanName)
	}
	ipv4 := info.IpV4Addresses[0]

	cidrStr := fmt.Sprintf("%s/%d", ipv4.Addr, ipv4.Netmask)
	_, cidr, perr := net.ParseCIDR(cidrStr)
	if perr != nil {
		return changed, fmt.Errorf("invalid CIDR '%s' for LAN '%s': %w", cidrStr, lanName, perr)
	}

	if entry.cidrString != cidrStr || entry.ipv4Addr != ipv4.Addr {
		changed = true
	}
	entry.cidr = cidr
	entry.ipv4Addr = ipv4.Addr
	entry.cidrString = cidrStr

	// Update IPv6 CIDR if available (non-fatal if absent)
	if ipv6, err := info.IpV6Addr(); err == nil {
		if entry.ipv6Addr != ipv6.Addr {
			changed = true
		}
		cidr6Str := fmt.Sprintf("%s/%d", ipv6.Addr, ipv6.PrefixLen)
		if _, cidr6, perr := net.ParseCIDR(cidr6Str); perr == nil {
			entry.cidr6 = cidr6
			entry.ipv6Addr = ipv6.Addr
		}
	}

	return changed, nil
}

// lanDevice returns the cached L3 device (e.g. "br-lan") for a registered LAN,
// or "" when the LAN is unknown or its device was never resolved. Reads only
// the registry — no UBUS call — so it is safe on the per-request path.
func lanDevice(lanName string) string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	if entry, ok := registry.byName[lanName]; ok {
		return entry.device
	}
	return ""
}

// =============================================================================
// LAN SETUP
// =============================================================================

func listenLanEvents(lan *NetworkLan) {
	ch := ubus.ListenInterface(lan.Name())
	for evt := range ch {
		netQueue.Exec("listenLanEvents", func() (any, error) {
			if evt.Event == ubus.IfEventDown && lan.Up() {
				lan.SetStatus(false)
			}

			if evt.Event == ubus.IfEventUp && !lan.Up() {
				time.Sleep(1000 * time.Millisecond) // add delay to wait for complete network bootup

				// Reinitialize TC only for captive interfaces. The flag is re-read
				// live so a toggle since boot is honored: a now-free interface must
				// NOT be re-shaped on ifup, and a now-captive one gets set up.
				if lanIsCaptive(lan.Name()) {
					if err := lan.ReinitializeTc(); err != nil {
						return nil, err
					}
				}

				// Rebuild CIDR cache — IP may have changed after reinit
				if _, err := updateLanCidr(lan.Name(), lan); err != nil {
					return nil, err
				}

				lan.SetStatus(true)

				// Re-apply portal config: this interface's IP (and possibly the
				// main LAN's IP) may have changed, so refresh the captive DNAT
				// target + split-horizon DNS. Best-effort — a failure here must not
				// undo the successful reinit above, so we deliberately ignore it
				// rather than fail the job (the TC reinit + CIDR cache are already
				// committed, and a portal reconcile retries on the next IP event).
				_ = ApplyPortalConfig()
			} else if evt.Event == ubus.IfEventUp {
				// ifup while already marked up: netifd re-announces an interface
				// without a preceding down when only its address changed (Apply
				// Changes → netifd reload), and a LAN that was registered IP-less at
				// boot starts as up (NewNetworkLan defaults up=true) so its first
				// ifup lands here too. The full reinit above is skipped, but the
				// CIDR cache MUST still track the new address — otherwise FindByIp
				// misclassifies every client on this LAN as unmanaged and the
				// portal funnel bounces them to /admin. Best-effort; only reconcile
				// the portal (DNAT target, portal IPs, split-horizon DNS) when the
				// address actually changed.
				if changed, err := updateLanCidr(lan.Name(), lan); err == nil && changed {
					_ = ApplyPortalConfig()
				}
			}

			return nil, nil
		})

	}
}

// SetupLanInterfaces registers every LAN candidate and starts one persistent
// event listener per LAN, but sets up traffic shaping (tc) ONLY for the captive
// interfaces. The captive set — driven by interfaces.json's EnableCaptivePortal
// flag (with the primary LAN captive by default) — is the single authority for
// what actually gets shaped + firewalled; a free interface is registered only so
// it can be identified and toggled captive live, and is otherwise left untouched
// (no tc, no custom nftables rules). Registering all candidates up front keeps
// the listener/registry set static so a later captive toggle only flips tc, with
// no goroutine or listener churn (see ReconcileInterfaces).
func SetupLanInterfaces() (err error) {
	ifaces, err := ubus.GetInterfaceNames()
	if err != nil {
		return err
	}

	cfg, err := config.ReadInterfacesConfig()
	if err != nil {
		return err
	}

	for _, ifname := range ifaces {
		if isNonLanInterface(ifname) {
			continue // WAN / loopback are never LAN candidates
		}

		// Resolve the L3 device so the primary-LAN default (br-lan) can be honored
		// for an interface with no explicit config entry.
		iface, ierr := ubus.GetNetworkInterface(ifname)
		if ierr != nil || iface == nil {
			continue
		}

		lan := NewNetworkLan(ifname)

		// Shape only captive interfaces; free ones are registered for identification
		// + live toggling but left otherwise untouched.
		if cfg.IsCaptivePortalEnabled(ifname, iface.Device) {
			if err = lan.SetupTrafficControl(); err != nil {
				return err
			}
		}

		go listenLanEvents(lan)

		if err = addLan(lan); err != nil {
			return err
		}
	}

	return nil
}

func getConfiguredLanNames(cfg config.BandwdCfg) []string {
	names := []string{}
	for name := range cfg.Lans {
		names = append(names, name)
	}
	return names
}

// =============================================================================
// PUBLIC API
// =============================================================================

// FindByIp returns the LAN whose subnet contains clientIp (IPv4 or IPv6).
// Optimized: parses the client IP once, then scans pre-parsed CIDRs under a
// read lock — no UBUS calls, no CIDR parsing, zero allocations on hit.
func FindByIp(clientIp string) (*NetworkLan, error) {
	ip := net.ParseIP(clientIp)
	if ip == nil {
		return nil, fmt.Errorf("invalid client IP: %s", clientIp)
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()

	for _, entry := range registry.byIp {
		// Check IPv4 CIDR
		if entry.cidr != nil && entry.cidr.Contains(ip) {
			return entry.lan, nil
		}
		// Check IPv6 CIDR
		if entry.cidr6 != nil && entry.cidr6.Contains(ip) {
			return entry.lan, nil
		}
	}

	return nil, fmt.Errorf("no matching LAN found for IP %s", clientIp)
}

// FindByName returns the LAN with the given interface name.
func FindByName(ifname string) (*NetworkLan, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	entry, ok := registry.byName[ifname]
	if !ok {
		return nil, errors.New("lan not found")
	}
	return entry.lan, nil
}

// FindAll returns all registered LAN instances.
func FindAll() []*NetworkLan {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	lans := make([]*NetworkLan, 0, registry.count)
	for _, entry := range registry.byIp {
		lans = append(lans, entry.lan)
	}
	return lans
}

// GetLanCount returns the number of registered LANs. O(1) — reads cached count.
func GetLanCount() int {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	return registry.count
}
