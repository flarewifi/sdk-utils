package hostfinder

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// DhcpLeaseInfo holds the information extracted from a DHCP lease entry
type DhcpLeaseInfo struct {
	Mac      string
	Hostname string
}

// FindHostFromDhcpLease searches DHCPv4 leases file by IP address.
// Returns MAC address and hostname if found in leases.
// Returns nil if IP not found in leases (caller should fallback to ARP).
// Returns error only if file cannot be read or has I/O issues.
//
// DHCPv4 lease format: <timestamp> <mac> <ipv4> <hostname> <client-id>
// Example: 1766499820 12:5e:9d:e1:6b:4d 10.0.15.7 debian 01:12:5e:9d:e1:6b:4d
func FindHostFromDhcpLease(ip string, leasePath string) (*DhcpLeaseInfo, error) {
	file, err := os.Open(leasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DHCP leases file %s: %w", leasePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		// dnsmasq lease format: <timestamp> <mac> <ip> <hostname> <client-id>
		// Need at least 4 fields to extract mac and hostname
		if len(fields) < 4 {
			continue // Skip malformed/incomplete lines
		}

		// Compare IP address (field index 2)
		if fields[2] == ip {
			return &DhcpLeaseInfo{
				Mac:      fields[1], // MAC is at index 1
				Hostname: fields[3], // hostname is at index 3
			}, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading DHCP leases: %w", err)
	}

	return nil, nil
}
