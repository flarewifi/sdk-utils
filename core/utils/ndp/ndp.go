// Package ndp reads the kernel IPv6 Neighbor Discovery Protocol (NDP) cache
// from /proc/net/ipv6_neigh, providing IP→MAC resolution for IPv6 clients.
// This is the IPv6 equivalent of the ARP table reader in core/utils/arp.
package ndp

import (
	"bufio"
	"net"
	"os"
	"strings"
)

// NeighTable maps IPv6 address strings → normalized lowercase MAC addresses.
type NeighTable map[string]string

// Table reads /proc/net/ipv6_neigh and returns the full neighbor cache.
// Entries with an invalid or all-zero MAC (INCOMPLETE / FAILED state) are
// skipped so callers never receive a placeholder MAC for an unresolved host.
// Returns nil if the file cannot be opened (e.g. kernel lacks IPv6 support).
//
// /proc/net/ipv6_neigh columns (space-separated):
//
//	ip_addr   devindex  state  flags  hwaddr             devname
//	2001:db8::1  3       64     0    aa:bb:cc:dd:ee:ff   br-lan
//
// "state" is a bitmask (NUD_REACHABLE=64, NUD_STALE=8, NUD_DELAY=4,
// NUD_PROBE=16, NUD_FAILED=32, NUD_INCOMPLETE=1).  We rely on hwaddr
// validity rather than state to filter bad entries.
func Table() NeighTable {
	f, err := os.Open("/proc/net/ipv6_neigh")
	if err != nil {
		return nil
	}
	defer f.Close()

	table := make(NeighTable)
	s := bufio.NewScanner(f)

	// Skip the header line
	s.Scan()

	for s.Scan() {
		fields := strings.Fields(s.Text())
		// Need at least 6 fields: ip_addr devindex state flags hwaddr devname
		if len(fields) < 6 {
			continue
		}

		ipStr := fields[0]
		hwAddr := fields[4]

		// Parse and validate the MAC address
		mac, err := net.ParseMAC(hwAddr)
		if err != nil {
			continue
		}

		// Skip all-zero MACs — these indicate an INCOMPLETE or FAILED
		// neighbor entry where the kernel has not yet resolved the address.
		if isZeroMAC(mac) {
			continue
		}

		// Normalize the IPv6 address to its canonical form for consistent
		// key lookup regardless of whether short or full notation was used.
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		table[ip.String()] = mac.String()
	}

	return table
}

// FindIpv6ByMac performs a reverse NDP lookup: given a MAC address, returns the
// first matching global-scope IPv6 address found in the neighbor cache, or an
// empty string if not found.  Link-local addresses (fe80::/10) are skipped.
func FindIpv6ByMac(mac string) string {
	// Normalize the input MAC via net.ParseMAC so that single-digit octets
	// (e.g. "a:b:c:d:e:f") are zero-padded to match the kernel's format
	// ("0a:0b:0c:0d:0e:0f") in /proc/net/ipv6_neigh.
	parsedInput, err := net.ParseMAC(mac)
	if err != nil {
		return ""
	}
	normalizedInput := strings.ToLower(parsedInput.String())

	f, err := os.Open("/proc/net/ipv6_neigh")
	if err != nil {
		return ""
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	s.Scan() // skip header

	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) < 6 {
			continue
		}

		hwAddr := strings.ToLower(fields[4])
		if hwAddr != normalizedInput {
			continue
		}

		// Validate MAC is non-zero
		parsed, err := net.ParseMAC(hwAddr)
		if err != nil || isZeroMAC(parsed) {
			continue
		}

		// Normalize and filter link-local addresses
		ip := net.ParseIP(fields[0])
		if ip == nil || ip.IsLinkLocalUnicast() {
			continue
		}

		return ip.String()
	}
	return ""
}

// Search looks up a MAC address for the given IPv6 address in the NDP cache.
// The ip parameter is normalized before lookup so short-form and full-form
// IPv6 addresses both match correctly.
// Returns the MAC address and true if found, or empty string and false if not.
func Search(ip string) (mac string, ok bool) {
	// Normalize the IP for consistent key lookup
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", false
	}
	table := Table()
	if table == nil {
		return "", false
	}
	mac, ok = table[parsed.String()]
	return mac, ok
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// isZeroMAC returns true if every byte of the MAC address is zero.
// The Linux kernel uses an all-zero MAC for NDP entries in INCOMPLETE or FAILED
// state, where the layer-2 address has not yet been resolved.
func isZeroMAC(mac net.HardwareAddr) bool {
	for _, b := range mac {
		if b != 0 {
			return false
		}
	}
	return true
}
