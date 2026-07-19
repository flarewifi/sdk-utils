package sdkutils

import (
	"fmt"
	"net"
)

// ValidateAndNormalizeMAC validates and normalizes a MAC address to lowercase colon-separated format.
// It accepts various MAC address formats (e.g., "AA:BB:CC:DD:EE:FF", "aa-bb-cc-dd-ee-ff", "aabbccddeeff")
// and returns a normalized lowercase colon-separated format (e.g., "aa:bb:cc:dd:ee:ff").
//
// Returns an error if the MAC address is empty, invalid, or consists of all zeroes.
func ValidateAndNormalizeMAC(mac string) (string, error) {
	if mac == "" {
		return "", fmt.Errorf("MAC address is empty")
	}

	// Parse MAC address using net.ParseMAC (handles various formats)
	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return "", fmt.Errorf("invalid MAC address format '%s': %v", mac, err)
	}

	// Check for all-zero MAC address (00:00:00:00:00:00)
	isAllZeroes := true
	for _, b := range hwAddr {
		if b != 0 {
			isAllZeroes = false
			break
		}
	}
	if isAllZeroes {
		return "", fmt.Errorf("MAC address cannot be all zeroes (00:00:00:00:00:00)")
	}

	// Return normalized lowercase colon-separated format (e.g., "aa:bb:cc:dd:ee:ff")
	return hwAddr.String(), nil
}

// ValidateIPAddress validates an IP address (IPv4 or IPv6) and returns it unchanged if valid.
// Returns an error if the IP address is empty or invalid.
func ValidateIPAddress(ip string) (string, error) {
	if ip == "" {
		return "", fmt.Errorf("IP address is empty")
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("invalid IP address format '%s'", ip)
	}

	// Return the original IP string (preserves user input format)
	return ip, nil
}

// ValidateIPv4Address validates an IPv4 address specifically.
// Returns an error if the IP address is empty, invalid, or not IPv4.
func ValidateIPv4Address(ip string) (string, error) {
	if ip == "" {
		return "", fmt.Errorf("IP address is empty")
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("invalid IP address format '%s'", ip)
	}

	// Check if it's an IPv4 address
	if parsedIP.To4() == nil {
		return "", fmt.Errorf("IP address '%s' is not IPv4", ip)
	}

	return ip, nil
}

// ValidateIPv6Address validates an IPv6 address specifically.
// Returns an error if the IP address is empty, invalid, or not IPv6.
func ValidateIPv6Address(ip string) (string, error) {
	if ip == "" {
		return "", fmt.Errorf("IP address is empty")
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("invalid IP address format '%s'", ip)
	}

	// Check if it's an IPv6 address
	if parsedIP.To4() != nil {
		return "", fmt.Errorf("IP address '%s' is not IPv6", ip)
	}

	return ip, nil
}

// GetIPVersion returns "ip" for IPv4 addresses, "ip6" for IPv6 addresses.
// Returns an error if the IP address is invalid.
// This is useful for nftables commands which require protocol family specification.
func GetIPVersion(ip string) (string, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("invalid IP address format '%s'", ip)
	}

	if parsedIP.To4() != nil {
		return "ip", nil
	}
	return "ip6", nil
}

// SeparatedIPs holds IPv4 and IPv6 addresses separated by version.
type SeparatedIPs struct {
	IPv4 []string
	IPv6 []string
}

// SeparateIPsByVersion takes a list of IP addresses, removes duplicates, and separates them into IPv4 and IPv6 slices.
// Duplicate IPs are automatically removed (first occurrence is kept).
// Returns an error if any IP address is invalid (fails the entire operation).
func SeparateIPsByVersion(ips []string) (SeparatedIPs, error) {
	result := SeparatedIPs{
		IPv4: []string{},
		IPv6: []string{},
	}

	// Deduplicate IPs first
	ips = SliceDedup(ips)

	for _, ip := range ips {
		version, err := GetIPVersion(ip)
		if err != nil {
			return SeparatedIPs{}, fmt.Errorf("invalid IP address in list: %v", err)
		}
		if version == "ip" {
			result.IPv4 = append(result.IPv4, ip)
		} else {
			result.IPv6 = append(result.IPv6, ip)
		}
	}
	return result, nil
}
