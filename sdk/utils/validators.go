package sdkutils

import (
	"fmt"
	"net"
)

// ValidateAndNormalizeMAC validates and normalizes a MAC address to lowercase colon-separated format.
// It accepts various MAC address formats (e.g., "AA:BB:CC:DD:EE:FF", "aa-bb-cc-dd-ee-ff", "aabbccddeeff")
// and returns a normalized lowercase colon-separated format (e.g., "aa:bb:cc:dd:ee:ff").
//
// Returns an error if the MAC address is empty or invalid.
func ValidateAndNormalizeMAC(mac string) (string, error) {
	if mac == "" {
		return "", fmt.Errorf("MAC address is empty")
	}

	// Parse MAC address using net.ParseMAC (handles various formats)
	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return "", fmt.Errorf("invalid MAC address format '%s': %v", mac, err)
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
