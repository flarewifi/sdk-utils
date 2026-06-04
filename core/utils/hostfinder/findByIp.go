//go:build !dev

package hostfinder

import (
	"errors"
	"fmt"
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
		return nil, err
	}

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
		leasePaths = []string{"/tmp/dhcp.leases"}
	}

	// Try to get MAC and hostname from DHCP leases first (authoritative source)
	var lastErr error
	for _, leasePath := range leasePaths {
		leaseInfo, err := FindHostFromDhcpLease(ip, leasePath)
		if err != nil {
			lastErr = err
			continue
		}

		if leaseInfo != nil {
			dhcpMAC := strings.ToUpper(leaseInfo.Mac)

			// SECURITY: Cross-validate DHCP MAC with ARP table
			arp.Search(ip)

			return &HostData{
				MacAddr:  dhcpMAC,
				IpAddr:   ip,
				Hostname: leaseInfo.Hostname,
			}, nil
		}
	}

	// If we had errors reading lease files, fallback to ARP
	if lastErr != nil {
		mac, ok := arp.Search(ip)
		if !ok {
			return nil, fmt.Errorf("failed to read DHCP leases (%w) and ARP lookup failed for IP: %s", lastErr, ip)
		}
		return &HostData{
			MacAddr:  strings.ToUpper(mac),
			IpAddr:   ip,
			Hostname: "",
		}, fmt.Errorf("failed to read DHCP leases: %w", lastErr)
	}

	// Not in any DHCP leases - fallback to ARP
	mac, ok := arp.Search(ip)
	if !ok {
		return nil, errors.New("cannot find host with IP: " + ip)
	}

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
	// Try DHCPv6/odhcpd lease files first (authoritative source)
	// odhcpd typically writes to /tmp/hosts/odhcpd or /tmp/odhcpd-leases
	dhcp6Paths := []string{
		"/tmp/hosts/odhcpd",
		"/tmp/odhcpd-leases",
	}

	var lastErr error
	for _, leasePath := range dhcp6Paths {
		leaseInfo, err := FindHostFromDhcp6Lease(ip, leasePath)
		if err != nil {
			lastErr = err
			continue
		}

		if leaseInfo != nil {
			// MAC may be empty if only odhcpd hostname info is present; fallback to NDP
			mac := strings.ToUpper(leaseInfo.Mac)
			if mac == "" {
				if ndpMAC, ok := ndp.Search(ip); ok {
					mac = strings.ToUpper(ndpMAC)
				}
			}

			if mac != "" {
				return &HostData{
					MacAddr:  mac,
					IpAddr:   ip,
					Hostname: leaseInfo.Hostname,
				}, nil
			}
		}
	}

	// Fallback to NDP cache (neighbor discovery, IPv6 equivalent of ARP)
	mac, ok := ndp.Search(ip)
	if !ok {
		if lastErr != nil {
			return nil, fmt.Errorf("DHCPv6 lease lookup failed (%w) and NDP lookup failed for IPv6: %s", lastErr, ip)
		}
		return nil, errors.New("cannot find host with IPv6: " + ip)
	}

	return &HostData{
		MacAddr:  strings.ToUpper(mac),
		IpAddr:   ip,
		Hostname: "",
	}, nil
}
