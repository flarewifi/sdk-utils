//go:build !dev

package nftables

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	jobque "core/utils/job-que"
	cmd "core/utils/shell"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	internetTable    string = "internet" // Our custom table
	tableFamily      string = "inet"     // inet family (handles both ipv4 and ipv6)
	forwardChain     string = "forward"
	preroutingChain  string = "prerouting"
	postroutingChain string = "postrouting"
	connMacMap       string = "connected_macs_map"
	connIpMap        string = "connected_ips_map"
	connIp6Map       string = "connected_ips6_map"
	connMacSet       string = "connected_macs_set"
)

var (
	nftMu  sync.RWMutex
	nftQue = jobque.NewJobQueue[any]()

	// ipToMac maps connected IP address (IPv4 or IPv6) → uppercase normalized MAC.
	// Populated on Connect, evicted on Disconnect.
	// Guarded by nftMu.
	ipToMac = make(map[string]string)

	// macToIps maps connected MAC → set of IPs registered for that device.
	// Populated on Connect, evicted on Disconnect.
	// Guarded by nftMu.
	macToIps = make(map[string]map[string]bool)
)

func Cleanup() {
	cmds := []string{
		// Delete our custom table (this removes all chains, maps, sets, and rules within it)
		fmt.Sprintf("nft delete table %s %s 2>/dev/null || true", tableFamily, internetTable),
	}
	cmd.ExecAll(cmds)
}

func Setup() (err error) {
	Cleanup()

	// Build nft batch script for atomic execution
	var batch strings.Builder

	// Create our custom internet table
	batch.WriteString(fmt.Sprintf("add table %s %s\n", tableFamily, internetTable))

	// Create custom forward and prerouting chains as base chains with hooks (priority -1 runs before fw4)
	batch.WriteString(fmt.Sprintf("add chain %s %s %s { type filter hook forward priority -250; policy drop; }\n", tableFamily, internetTable, forwardChain))
	batch.WriteString(fmt.Sprintf("add chain %s %s %s { type nat hook prerouting priority -1; policy accept; }\n", tableFamily, internetTable, preroutingChain))

	// Create maps and sets in our custom table
	// IPv4 verdict map (download traffic: accept + accounting)
	batch.WriteString(fmt.Sprintf("add map %s %s %s { type ipv4_addr : verdict ; counter; }\n", tableFamily, internetTable, connIpMap))
	// IPv6 verdict map (download traffic: accept + accounting)
	batch.WriteString(fmt.Sprintf("add map %s %s %s { type ipv6_addr : verdict ; counter; }\n", tableFamily, internetTable, connIp6Map))
	// MAC verdict map and set (protocol-agnostic)
	batch.WriteString(fmt.Sprintf("add map %s %s %s { type ether_addr : verdict ; counter; }\n", tableFamily, internetTable, connMacMap))
	batch.WriteString(fmt.Sprintf("add set %s %s %s { type ether_addr; }\n", tableFamily, internetTable, connMacSet))

	// Create postrouting chain for anti-tethering (TTL set)
	// Sets outgoing TTL to 1 so tethered devices cannot forward packets (TTL drops to 0)
	batch.WriteString(fmt.Sprintf("add chain %s %s %s { type filter hook postrouting priority 0; policy accept; }\n", tableFamily, internetTable, postroutingChain))

	// Add rules to our custom forward chain.
	//
	// Verdict-map-only design: no conntrack lookups, O(1) hash table matches.
	// This avoids per-packet conntrack overhead that causes latency for gaming.
	//
	// Rule order (first terminal verdict wins):
	//   1. Upload: source MAC verdict map — accept if MAC is registered.
	//   2. Download (IPv4): destination IP verdict map — accept + count.
	//   3. Download (IPv6): destination IP6 verdict map — accept + count.
	//   4. (implicit) chain policy drop — everything else is blocked.
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ether saddr vmap @%s\n", tableFamily, internetTable, forwardChain, connMacMap))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ip daddr vmap @%s\n", tableFamily, internetTable, forwardChain, connIpMap))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ip6 daddr vmap @%s\n", tableFamily, internetTable, forwardChain, connIp6Map))
	// Service port jumps will be appended after this rule for unauthenticated clients.

	// Execute batch command using nft -f - with heredoc for atomic execution
	nftCmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
	err = cmd.Exec(nftCmd, nil)
	if err != nil {
		return err
	}

	return nil
}

// SetupCaptivePortal installs prerouting DNAT rules for a LAN interface.
// routerIp4 is the LAN-facing IPv4 address; routerIp6 is the LAN-facing IPv6
// address (may be empty if the interface has no global IPv6 address yet).
// Link-local IPv6 addresses must NOT be passed as routerIp6 — they require
// interface-scoped routing that nftables DNAT does not support directly.
func SetupCaptivePortal(dev string, routerIp4 string, routerIp6 string) (err error) {
	// Guard: reject link-local IPv6 addresses — DNAT to link-local requires
	// an interface scope that nftables cannot represent in the rule syntax.
	if routerIp6 != "" {
		parsed := net.ParseIP(routerIp6)
		if parsed == nil || parsed.IsLinkLocalUnicast() {
			log.Printf("[WARN] SetupCaptivePortal: ignoring link-local/invalid IPv6 address %q for dev %s", routerIp6, dev)
			routerIp6 = ""
		}
	}

	contextInfo := fmt.Sprintf("Device=%s, RouterIPv4=%s, RouterIPv6=%s", dev, routerIp4, routerIp6)

	_, err = nftQue.ExecWithTimeout(
		30*time.Second,
		"Setup Captive Portal",
		contextInfo,
		func() (any, error) {
			// Build nft batch script for atomic execution
			var batch strings.Builder

			// Allow already authenticated devices to bypass captive portal
			batch.WriteString(fmt.Sprintf("add rule %s %s %s ether saddr @%s counter accept\n", tableFamily, internetTable, preroutingChain, connMacSet))

			// Redirect plain HTTP (port 80) to the captive portal (IPv4).
			// Port 443 is intentionally NOT intercepted: MITM'ing TLS breaks the
			// browser. Modern OSes are instead pointed at the portal via the RFC
			// 8910 advertisement (DHCP option 114); port 80 stays as the legacy
			// detection fallback for clients that still probe over HTTP.
			if routerIp4 != "" {
				batch.WriteString(fmt.Sprintf("add rule %s %s %s iif %s tcp dport { 80 } counter dnat ip to %s\n", tableFamily, internetTable, preroutingChain, dev, routerIp4))
			}

			// Redirect plain HTTP (port 80) to the captive portal (IPv6).
			if routerIp6 != "" {
				batch.WriteString(fmt.Sprintf("add rule %s %s %s iif %s tcp dport { 80 } counter dnat ip6 to %s\n", tableFamily, internetTable, preroutingChain, dev, routerIp6))
			}

			// Anti-tethering: set TTL=1 on IPv4 packets going out through this LAN device
			if routerIp4 != "" {
				batch.WriteString(fmt.Sprintf("add rule %s %s %s oifname %s ip ttl set 1\n", tableFamily, internetTable, postroutingChain, dev))
			}

			// Anti-tethering: set hop limit=1 on IPv6 packets going out through this LAN device
			if routerIp6 != "" {
				batch.WriteString(fmt.Sprintf("add rule %s %s %s oifname %s ip6 hoplimit set 1\n", tableFamily, internetTable, postroutingChain, dev))
			}

			// Execute batch command using nft -f - with heredoc for atomic execution
			nftCmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
			return nil, cmd.Exec(nftCmd, nil)
		},
	)
	return err
}

func Connect(ip string, mac string) error {
	contextInfo := fmt.Sprintf("IP=%s, MAC=%s", ip, mac)

	_, err := nftQue.ExecWithTimeout(
		30*time.Second,
		"Connect Device",
		contextInfo,
		func() (any, error) {
			err := doConnect(ip, mac)
			return nil, err
		},
	)
	return err
}

func Disconnect(ip string, mac string) error {
	contextInfo := fmt.Sprintf("IP=%s, MAC=%s", ip, mac)

	_, err := nftQue.ExecWithTimeout(
		30*time.Second,
		"Disconnect Device",
		contextInfo,
		func() (any, error) {
			err := doDisconnect(ip, mac)
			return nil, err
		},
	)
	return err
}

func IsConnected(mac string) bool {
	contextInfo := fmt.Sprintf("MAC=%s", mac)

	result, err := nftQue.ExecWithTimeout(
		30*time.Second,
		"Check Connection Status",
		contextInfo,
		func() (any, error) {
			return isConnected(mac), nil
		},
	)

	if err != nil {
		log.Printf("[ERROR] IsConnected check failed for %s: %v", mac, err)
		return false
	}

	return result.(bool)
}

// GetMacByIp returns the normalized uppercase MAC address for a currently
// connected IP address (IPv4 or IPv6), or an empty string if the IP is not
// in the cache. The cache is populated on Connect and evicted on Disconnect,
// so it only contains entries for devices actively allowed through the firewall.
func GetMacByIp(ip string) string {
	nftMu.RLock()
	defer nftMu.RUnlock()
	return ipToMac[ip]
}

// GetMacsByIps returns a map of IP→MAC for a batch of IP addresses, acquiring
// the lock only once. This is more efficient than calling GetMacByIp in a loop.
// IPs not found in the cache are omitted from the result map.
func GetMacsByIps(ips []string) map[string]string {
	nftMu.RLock()
	defer nftMu.RUnlock()

	result := make(map[string]string, len(ips))
	for _, ip := range ips {
		if mac := ipToMac[ip]; mac != "" {
			result[ip] = mac
		}
	}
	return result
}

// isIPv6 returns true if ip is a valid IPv6 address (not IPv4 or IPv4-mapped).
func isIPv6(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.To4() == nil
}

func isConnected(mac string) bool {
	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		log.Printf("[ERROR] isConnected: invalid MAC address '%s': %v", mac, err)
		return false
	}

	err = cmd.Exec(fmt.Sprintf("nft get element %s %s %s '{ %s }'", tableFamily, internetTable, connMacSet, normalizedMAC), nil)
	return err == nil
}

func doConnect(ip string, mac string) error {
	// Validate IP address
	if _, err := sdkutils.ValidateIPAddress(ip); err != nil {
		return fmt.Errorf("invalid IP address: %v", err)
	}

	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %v", err)
	}

	// Choose the correct IP verdict map based on IP version (used for download accounting only).
	ipMap := connIpMap
	if isIPv6(ip) {
		ipMap = connIp6Map
	}

	// Step 1: Add this IP to the download-accounting verdict map.
	// Idempotent via || true — safe if called twice for the same IP.
	cmd.Exec(fmt.Sprintf("nft add element %s %s %s '{ %s : accept }' 2>/dev/null || true", tableFamily, internetTable, ipMap, ip), nil)

	// Step 2: Add MAC to the upload-accounting verdict map and the allow set.
	// These are idempotent — the second call for a dual-stack device is a no-op.
	cmd.Exec(fmt.Sprintf("nft add element %s %s %s '{ %s : accept }' 2>/dev/null || true", tableFamily, internetTable, connMacMap, normalizedMAC), nil)
	cmd.Exec(fmt.Sprintf("nft add element %s %s %s '{ %s }' 2>/dev/null || true", tableFamily, internetTable, connMacSet, normalizedMAC), nil)

	// Record IP→MAC and MAC→IPs mappings for traffic accounting and GetMacByIp().
	nftMu.Lock()
	ipToMac[ip] = normalizedMAC
	if macToIps[normalizedMAC] == nil {
		macToIps[normalizedMAC] = make(map[string]bool)
	}
	macToIps[normalizedMAC][ip] = true
	nftMu.Unlock()

	return nil
}

func doDisconnect(ip string, mac string) error {
	// Validate IP address
	if _, err := sdkutils.ValidateIPAddress(ip); err != nil {
		return fmt.Errorf("invalid IP address: %v", err)
	}

	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %v", err)
	}

	// Choose the correct IP verdict map based on IP version.
	ipMap := connIpMap
	if isIPv6(ip) {
		ipMap = connIp6Map
	}

	// Step 1: Remove this IP from the download-accounting verdict map.
	// Best-effort — if it was never added (e.g. partial connect failure), swallow.
	cmd.Exec(fmt.Sprintf("nft delete element %s %s %s '{ %s : accept }' 2>/dev/null || true", tableFamily, internetTable, ipMap, ip), nil)

	// Step 2: Update in-memory maps.  Only remove MAC-level entries when all
	// IPs for this device have been disconnected (handles dual-stack correctly).
	nftMu.Lock()
	delete(ipToMac, ip)
	remainingIPs := 0
	if ips, ok := macToIps[normalizedMAC]; ok {
		delete(ips, ip)
		remainingIPs = len(ips)
		if remainingIPs == 0 {
			delete(macToIps, normalizedMAC)
		}
	}
	nftMu.Unlock()

	// Step 3: Once all IPs for this MAC are gone, remove the MAC from the
	// nftables allow set/map and flush conntrack entries so existing connections
	// are cut immediately rather than left alive until they time out naturally.
	if remainingIPs == 0 {
		cmd.Exec(fmt.Sprintf("nft delete element %s %s %s '{ %s : accept }' 2>/dev/null || true", tableFamily, internetTable, connMacMap, normalizedMAC), nil)
		cmd.Exec(fmt.Sprintf("nft delete element %s %s %s '{ %s }' 2>/dev/null || true", tableFamily, internetTable, connMacSet, normalizedMAC), nil)
		// Flush conntrack entries so the session cutoff is immediate.
		// conntrack may not be present on all OpenWRT images; ignore errors.
		cmd.Exec(fmt.Sprintf("conntrack -D --orig-mac-src %s 2>/dev/null || true", normalizedMAC), nil)
	}

	return nil
}
