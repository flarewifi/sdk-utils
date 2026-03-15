//go:build !dev

package hostfinder

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"core/internal/modules/uci"
	"core/utils/arp"
	"core/utils/ndp"
)

func GetHostFromRequest(r *http.Request) (*HostData, error) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Printf("[HostFinder] ERROR: Failed to split host:port from RemoteAddr=%s: %v", r.RemoteAddr, err)
		return nil, err
	}
	log.Printf("[HostFinder] DEBUG: Extracted IP=%s from RemoteAddr=%s", ip, r.RemoteAddr)

	// Detect IP version to route to the correct lookup path
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}
	isIPv6 := parsedIP.To4() == nil

	if isIPv6 {
		return getHostFromIPv6(ip)
	}
	return getHostFromIPv4(ip)
}

// =============================================================================
// IPv4 HOST LOOKUP
// =============================================================================

func getHostFromIPv4(ip string) (*HostData, error) {
	// Get DHCPv4 lease file paths from UCI configuration
	dhcpApi := uci.NewUciDhcpApi()
	leasePaths, err := dhcpApi.GetDnsmasqLeasesFiles()
	if err != nil {
		// Shouldn't happen since GetDnsmasqLeasesFiles always returns nil error with default
		log.Printf("[HostFinder] WARN: Failed to get DHCP lease paths, using default: %v", err)
		leasePaths = []string{"/tmp/dhcp.leases"}
	}
	log.Printf("[HostFinder] DEBUG: Checking DHCP lease files: %v", leasePaths)

	// Try to get MAC and hostname from DHCP leases first (authoritative source)
	var lastErr error
	for _, leasePath := range leasePaths {
		log.Printf("[HostFinder] DEBUG: Searching for IP=%s in lease file: %s", ip, leasePath)
		leaseInfo, err := FindHostFromDhcpLease(ip, leasePath)
		if err != nil {
			log.Printf("[HostFinder] WARN: Error reading lease file %s: %v", leasePath, err)
			lastErr = err
			continue
		}

		if leaseInfo != nil {
			dhcpMAC := strings.ToUpper(leaseInfo.Mac)
			log.Printf("[HostFinder] SUCCESS: Found in DHCP leases - IP=%s, MAC=%s, Hostname=%s", ip, dhcpMAC, leaseInfo.Hostname)

			// SECURITY: Cross-validate DHCP MAC with ARP table
			if arpMAC, ok := arp.Search(ip); ok {
				arpMACUpper := strings.ToUpper(arpMAC)
				if arpMACUpper != dhcpMAC {
					log.Printf("[SECURITY] HostFinder: DHCP/ARP MAC mismatch for IP=%s - DHCP=%s, ARP=%s (possible DHCP spoof or stale lease)", ip, dhcpMAC, arpMACUpper)
				}
			}

			return &HostData{
				MacAddr:  dhcpMAC,
				IpAddr:   ip,
				Hostname: leaseInfo.Hostname,
			}, nil
		}
		log.Printf("[HostFinder] DEBUG: IP=%s not found in lease file: %s", ip, leasePath)
	}

	// If we had errors reading lease files, fallback to ARP
	if lastErr != nil {
		log.Printf("[HostFinder] WARN: DHCP lease lookup failed, falling back to ARP for IP=%s", ip)
		mac, ok := arp.Search(ip)
		if !ok {
			log.Printf("[HostFinder] ERROR: ARP lookup also failed for IP=%s", ip)
			return nil, fmt.Errorf("failed to read DHCP leases (%w) and ARP lookup failed for IP: %s", lastErr, ip)
		}
		log.Printf("[HostFinder] SUCCESS: Found via ARP fallback - IP=%s, MAC=%s (no hostname)", ip, mac)
		return &HostData{
			MacAddr:  strings.ToUpper(mac),
			IpAddr:   ip,
			Hostname: "",
		}, fmt.Errorf("failed to read DHCP leases: %w", lastErr)
	}

	// Not in any DHCP leases - fallback to ARP
	log.Printf("[HostFinder] DEBUG: IP=%s not in any DHCP leases, trying ARP lookup", ip)
	mac, ok := arp.Search(ip)
	if !ok {
		log.Printf("[HostFinder] ERROR: ARP lookup failed for IP=%s", ip)
		return nil, errors.New("cannot find host with IP: " + ip)
	}

	log.Printf("[HostFinder] SUCCESS: Found via ARP - IP=%s, MAC=%s (no hostname)", ip, mac)
	return &HostData{
		MacAddr:  strings.ToUpper(mac),
		IpAddr:   ip,
		Hostname: "",
	}, nil
}

// =============================================================================
// IPv6 HOST LOOKUP
// =============================================================================

func getHostFromIPv6(ip string) (*HostData, error) {
	log.Printf("[HostFinder] DEBUG: IPv6 lookup for IP=%s", ip)

	// Try DHCPv6/odhcpd lease files first (authoritative source)
	// odhcpd typically writes to /tmp/hosts/odhcpd or /tmp/odhcpd-leases
	dhcp6Paths := []string{
		"/tmp/hosts/odhcpd",
		"/tmp/odhcpd-leases",
	}

	var lastErr error
	for _, leasePath := range dhcp6Paths {
		log.Printf("[HostFinder] DEBUG: Searching for IPv6=%s in DHCPv6 lease file: %s", ip, leasePath)
		leaseInfo, err := FindHostFromDhcp6Lease(ip, leasePath)
		if err != nil {
			log.Printf("[HostFinder] WARN: Error reading DHCPv6 lease file %s: %v", leasePath, err)
			lastErr = err
			continue
		}

		if leaseInfo != nil {
			// MAC may be empty if only odhcpd hostname info is present; fallback to NDP
			mac := strings.ToUpper(leaseInfo.Mac)
			if mac == "" {
				log.Printf("[HostFinder] DEBUG: DHCPv6 lease found hostname but no MAC for IPv6=%s, trying NDP", ip)
				if ndpMAC, ok := ndp.Search(ip); ok {
					mac = strings.ToUpper(ndpMAC)
				}
			}

			if mac != "" {
				log.Printf("[HostFinder] SUCCESS: Found via DHCPv6 - IPv6=%s, MAC=%s, Hostname=%s", ip, mac, leaseInfo.Hostname)
				return &HostData{
					MacAddr:  mac,
					IpAddr:   ip,
					Hostname: leaseInfo.Hostname,
				}, nil
			}
		}
		log.Printf("[HostFinder] DEBUG: IPv6=%s not found in DHCPv6 lease file: %s", ip, leasePath)
	}

	// Fallback to NDP cache (neighbor discovery, IPv6 equivalent of ARP)
	log.Printf("[HostFinder] DEBUG: Trying NDP cache for IPv6=%s", ip)
	mac, ok := ndp.Search(ip)
	if !ok {
		if lastErr != nil {
			log.Printf("[HostFinder] ERROR: DHCPv6 lease errors and NDP lookup failed for IPv6=%s", ip)
			return nil, fmt.Errorf("DHCPv6 lease lookup failed (%w) and NDP lookup failed for IPv6: %s", lastErr, ip)
		}
		log.Printf("[HostFinder] ERROR: NDP lookup failed for IPv6=%s", ip)
		return nil, errors.New("cannot find host with IPv6: " + ip)
	}

	log.Printf("[HostFinder] SUCCESS: Found via NDP - IPv6=%s, MAC=%s (no hostname)", ip, mac)
	return &HostData{
		MacAddr:  strings.ToUpper(mac),
		IpAddr:   ip,
		Hostname: "",
	}, nil
}
