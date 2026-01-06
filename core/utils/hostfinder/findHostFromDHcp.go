package hostfinder

import (
	"bufio"
	"fmt"
	"log"
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
	log.Printf("[DHCP Lease] DEBUG: Opening lease file: %s", leasePath)
	file, err := os.Open(leasePath)
	if err != nil {
		log.Printf("[DHCP Lease] ERROR: Failed to open %s: %v", leasePath, err)
		return nil, fmt.Errorf("failed to open DHCP leases file %s: %w", leasePath, err)
	}
	defer file.Close()

	lineCount := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		fields := strings.Fields(line)

		// dnsmasq lease format: <timestamp> <mac> <ip> <hostname> <client-id>
		// Need at least 4 fields to extract mac and hostname
		if len(fields) < 4 {
			log.Printf("[DHCP Lease] DEBUG: Skipping malformed line %d (fields=%d): %s", lineCount, len(fields), line)
			continue // Skip malformed/incomplete lines
		}

		// Compare IP address (field index 2)
		if fields[2] == ip {
			log.Printf("[DHCP Lease] SUCCESS: Found IP %s in %s at line %d - MAC=%s, Hostname=%s, Line: %s",
				ip, leasePath, lineCount, fields[1], fields[3], line)
			return &DhcpLeaseInfo{
				Mac:      fields[1], // MAC is at index 1
				Hostname: fields[3], // hostname is at index 3
			}, nil
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[DHCP Lease] ERROR: Scanner error reading %s: %v", leasePath, err)
		return nil, fmt.Errorf("error reading DHCP leases: %w", err)
	}

	// IP not found in leases - return nil (not an error)
	log.Printf("[DHCP Lease] DEBUG: IP %s not found in %s (scanned %d lines)", ip, leasePath, lineCount)
	return nil, nil
}
