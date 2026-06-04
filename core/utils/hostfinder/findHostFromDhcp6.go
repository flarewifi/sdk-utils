package hostfinder

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

// extractMACFromDUID attempts to extract a 6-byte Ethernet MAC address from a
// DHCPv6 DUID string.  Common DUID types that embed a MAC address:
//   - DUID-LL  (type 3): "00:03:00:01:<mac6bytes>"     → 10 colon-separated octets
//   - DUID-LLT (type 1): "00:01:00:01:<timestamp>:<mac>" → 14+ colon-separated octets
//   - Plain MAC: exactly 6 colon-separated octets (dnsmasq occasionally writes these)
//
// Returns an empty string if the DUID does not contain a recoverable MAC.
func extractMACFromDUID(duid string) string {
	parts := strings.Split(duid, ":")
	switch len(parts) {
	case 6:
		// Plain 6-byte MAC address
		if mac, err := net.ParseMAC(duid); err == nil {
			return mac.String()
		}
	case 10:
		// DUID-LL: "00:03:00:01" + 6 MAC bytes
		if parts[0] == "00" && parts[1] == "03" {
			macStr := strings.Join(parts[4:], ":")
			if mac, err := net.ParseMAC(macStr); err == nil {
				return mac.String()
			}
		}
	case 14:
		// DUID-LLT: "00:01:00:01" + 4 timestamp bytes + 6 MAC bytes
		if parts[0] == "00" && parts[1] == "01" {
			macStr := strings.Join(parts[8:], ":")
			if mac, err := net.ParseMAC(macStr); err == nil {
				return mac.String()
			}
		}
	}
	return ""
}

// Dhcp6LeaseInfo holds information extracted from a DHCPv6/SLAAC lease entry.
type Dhcp6LeaseInfo struct {
	Mac      string
	Hostname string
}

// FindHostFromDhcp6Lease searches odhcpd IPv6 lease files by IPv6 address.
// Returns MAC address and hostname if found.
// Returns nil if IP not found (caller should fallback to NDP).
// Returns error only if the file cannot be read.
//
// odhcpd lease format (space-separated fields):
//
//	# <iface> <duid> <iaid> <name> <lifetime> <assigned_addr/prefix>
//	br-lan - 1 myhost 86400 2001:db8::1 aa:bb:cc:dd:ee:ff
//
// The exact format can vary by OpenWRT version. We support two common layouts:
//
//  1. dnsmasq-style DHCPv6: <timestamp> <duid/mac> <ipv6> <hostname> ...
//  2. odhcpd-style:         # <iface> <duid> <iaid> <hostname> <lifetime> <ipv6>
func FindHostFromDhcp6Lease(ip string, leasePath string) (*Dhcp6LeaseInfo, error) {
	// Normalize the search IP for consistent comparison
	searchIP := net.ParseIP(ip)
	if searchIP == nil {
		return nil, fmt.Errorf("invalid IPv6 address: %s", ip)
	}
	normalizedSearchIP := searchIP.String()

	file, err := os.Open(leasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DHCPv6 leases file %s: %w", leasePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// odhcpd format starts with '#': # <iface> <duid> <iaid> <hostname> <lifetime> <ipv6> [<mac>]
		if strings.HasPrefix(line, "#") {
			fields := strings.Fields(line[1:]) // strip leading '#'
			// Minimum 6 fields: iface duid iaid hostname lifetime ipv6
			if len(fields) < 6 {
				continue
			}
			// IPv6 is field index 5
			leaseIP := net.ParseIP(fields[5])
			if leaseIP == nil {
				continue
			}
			if leaseIP.String() != normalizedSearchIP {
				continue
			}
			hostname := fields[3]
			if hostname == "-" {
				hostname = ""
			}
			// MAC may appear as field 7 in newer odhcpd versions
			mac := ""
			if len(fields) >= 8 {
				mac = fields[7]
			}
			return &Dhcp6LeaseInfo{Mac: mac, Hostname: hostname}, nil
		}

		// dnsmasq-style DHCPv6: <timestamp> <duid-or-mac> <ipv6> <hostname> ...
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		leaseIP := net.ParseIP(fields[2])
		if leaseIP == nil {
			continue
		}
		if leaseIP.String() != normalizedSearchIP {
			continue
		}
		// fields[1] may be a full DUID (e.g. "00:03:00:01:aa:bb:cc:dd:ee:ff")
		// rather than a plain 6-byte MAC.  Extract the embedded MAC if possible.
		mac := extractMACFromDUID(fields[1])
		hostname := fields[3]
		if hostname == "*" || hostname == "-" {
			hostname = ""
		}
		return &Dhcp6LeaseInfo{Mac: mac, Hostname: hostname}, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading DHCPv6 leases: %w", err)
	}

	return nil, nil
}
