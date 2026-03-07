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
)

func GetHostFromRequest(r *http.Request) (*HostData, error) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Printf("[HostFinder] ERROR: Failed to split host:port from RemoteAddr=%s: %v", r.RemoteAddr, err)
		return nil, err
	}
	log.Printf("[HostFinder] DEBUG: Extracted IP=%s from RemoteAddr=%s", ip, r.RemoteAddr)

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
	// Iterate through all configured lease files
	var lastErr error
	for _, leasePath := range leasePaths {
		log.Printf("[HostFinder] DEBUG: Searching for IP=%s in lease file: %s", ip, leasePath)
		leaseInfo, err := FindHostFromDhcpLease(ip, leasePath)
		if err != nil {
			// Error reading this lease file - try next one
			log.Printf("[HostFinder] WARN: Error reading lease file %s: %v", leasePath, err)
			lastErr = err
			continue
		}

		// If found in DHCP leases, use that data (authoritative)
		if leaseInfo != nil {
			dhcpMAC := strings.ToUpper(leaseInfo.Mac)
			log.Printf("[HostFinder] SUCCESS: Found in DHCP leases - IP=%s, MAC=%s, Hostname=%s", ip, dhcpMAC, leaseInfo.Hostname)

			// SECURITY: Cross-validate DHCP MAC with ARP table
			// If they disagree, it could indicate DHCP spoofing or stale lease
			// Log the discrepancy but trust DHCP as the authoritative source
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

	// If we had errors reading lease files, fallback to ARP and return error for logging
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

	// Not in any DHCP leases - fallback to ARP for MAC address
	log.Printf("[HostFinder] DEBUG: IP=%s not in any DHCP leases, trying ARP lookup", ip)
	mac, ok := arp.Search(ip)
	if !ok {
		log.Printf("[HostFinder] ERROR: ARP lookup failed for IP=%s", ip)
		return nil, errors.New("cannot find host with IP: " + ip)
	}

	// Return with MAC from ARP, empty hostname
	log.Printf("[HostFinder] SUCCESS: Found via ARP - IP=%s, MAC=%s (no hostname)", ip, mac)
	return &HostData{
		MacAddr:  strings.ToUpper(mac),
		IpAddr:   ip,
		Hostname: "",
	}, nil
}
