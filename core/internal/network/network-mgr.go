package network

import (
	"errors"
	"fmt"
	"log"
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
	cidr       *net.IPNet // pre-parsed once during addLan()
	ipv4Addr   string     // e.g. "192.168.1.1"
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

// addLan fetches the LAN's IPv4 address, parses its CIDR once, and stores the
// pre-built entry in the registry. Called once per LAN at startup.
func addLan(lan *NetworkLan) error {
	ipv4, err := lan.GetInterface().IpV4Addr()
	if err != nil {
		return fmt.Errorf("failed to get IPv4 for LAN '%s': %w", lan.Name(), err)
	}

	cidrStr := fmt.Sprintf("%s/%d", ipv4.Addr, ipv4.Netmask)
	_, cidr, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return fmt.Errorf("invalid CIDR '%s' for LAN '%s': %w", cidrStr, lan.Name(), err)
	}

	entry := &lanEntry{
		name:       lan.Name(),
		lan:        lan,
		cidr:       cidr,
		ipv4Addr:   ipv4.Addr,
		cidrString: cidrStr,
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.byName[lan.Name()] = entry
	registry.byIp = append(registry.byIp, entry)
	registry.count++

	return nil
}

// updateLanCidr rebuilds the cached CIDR for a LAN whose IP may have changed
// (e.g. after an interface reinit). Called from listenLanEvents on IfEventUp.
func updateLanCidr(lanName string, lan *NetworkLan) error {
	ipv4, err := lan.GetInterface().IpV4Addr()
	if err != nil {
		return fmt.Errorf("failed to get IPv4 for LAN '%s': %w", lanName, err)
	}

	cidrStr := fmt.Sprintf("%s/%d", ipv4.Addr, ipv4.Netmask)
	_, cidr, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return fmt.Errorf("invalid CIDR '%s' for LAN '%s': %w", cidrStr, lanName, err)
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	entry, ok := registry.byName[lanName]
	if !ok {
		return fmt.Errorf("LAN '%s' not found in registry", lanName)
	}

	entry.cidr = cidr
	entry.ipv4Addr = ipv4.Addr
	entry.cidrString = cidrStr

	return nil
}

// =============================================================================
// LAN SETUP
// =============================================================================

func listenLanEvents(lan *NetworkLan) {
	ch := ubus.ListenInterface(lan.Name())
	for evt := range ch {
		netQueue.Exec("listenLanEvents", func() (any, error) {
			if evt.Event == ubus.IfEventDown && lan.Up() {
				log.Printf("LAN interface '%s' went DOWN", lan.Name())
				lan.SetStatus(false)
			}

			if evt.Event == ubus.IfEventUp && !lan.Up() {
				log.Printf("LAN interface '%s' came UP, reinitializing...", lan.Name())
				time.Sleep(1000 * time.Millisecond) // add delay to wait for complete network bootup

				// Reinitialize TC (handles IP changes and ensures proper setup)
				err := lan.ReinitializeTc()
				if err != nil {
					log.Printf("ERROR: Failed to reinitialize TC for LAN '%s': %v", lan.Name(), err)
					return nil, err
				}

				// Rebuild CIDR cache — IP may have changed after reinit
				if err := updateLanCidr(lan.Name(), lan); err != nil {
					log.Printf("ERROR: Failed to update CIDR cache for LAN '%s': %v", lan.Name(), err)
					return nil, err
				}

				lan.SetStatus(true)
				log.Printf("LAN interface '%s' reinitialized successfully", lan.Name())
			}

			return nil, nil
		})

		log.Println("Interface event: ", evt)
	}
}

func SetupLanInterfaces() (err error) {
	log.Println("SetupLanInterfaces: Starting LAN interface setup...")

	ifaces, err := ubus.GetInterfaceNames()
	log.Println("ubus.GetNetworkInterfaces(): ", ifaces)
	if err != nil {
		log.Printf("ERROR: Failed to get interface names from UBUS: %v", err)
		return err
	}

	cfg, err := config.ReadBandwidthConfig()
	if err != nil {
		log.Printf("ERROR: Failed to read bandwidth config: %v", err)
		return err
	}
	log.Printf("Bandwidth config loaded. Configured LANs: %v", getConfiguredLanNames(cfg))

	lanCount := 0
	for _, ifname := range ifaces {
		_, ok := cfg.Lans[ifname]
		if ok {
			lan := NewNetworkLan(ifname)

			err = lan.SetupTrafficControl()
			if err != nil {
				log.Printf("ERROR: Failed to setup traffic control for interface %s: %v", ifname, err)
				return err
			}
			go listenLanEvents(lan)

			if err = addLan(lan); err != nil {
				log.Printf("ERROR: Failed to add LAN '%s' to registry: %v", ifname, err)
				return err
			}
			lanCount++
			log.Printf("LAN interface '%s' added to registry", ifname)
		} else {
			log.Printf("Interface '%s' not found in bandwidth config, skipping", ifname)
		}
	}

	log.Printf("SetupLanInterfaces complete: %d LAN(s) configured", lanCount)

	if lanCount == 0 {
		log.Println("WARNING: No LAN interfaces were configured! Check bandwidth.json config.")
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

// FindByIp returns the LAN whose subnet contains clientIp.
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
		if entry.cidr.Contains(ip) {
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
