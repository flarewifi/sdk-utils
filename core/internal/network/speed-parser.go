package network

import (
	"strconv"
	"strings"
)

// DefaultLinkSpeed is the fallback speed in Mbps when link speed cannot be detected
const DefaultLinkSpeed int = 1000

// ParseLinkSpeed parses a UBUS speed string and returns the speed in Mbps.
// Supported formats: "1000M", "10000M", "100M", "1000", "10G", etc.
// Returns DefaultLinkSpeed (1000 Mbps) if parsing fails or string is empty.
func ParseLinkSpeed(speedStr string) int {
	if speedStr == "" || speedStr == "-" || strings.ToLower(speedStr) == "unknown" {
		return DefaultLinkSpeed
	}

	speedStr = strings.TrimSpace(speedStr)
	speedStr = strings.ToUpper(speedStr)

	var multiplier int = 1
	var numStr string

	// Handle suffix-based formats
	if strings.HasSuffix(speedStr, "G") {
		multiplier = 1000
		numStr = strings.TrimSuffix(speedStr, "G")
	} else if strings.HasSuffix(speedStr, "M") {
		multiplier = 1
		numStr = strings.TrimSuffix(speedStr, "M")
	} else if strings.HasSuffix(speedStr, "GBIT") {
		multiplier = 1000
		numStr = strings.TrimSuffix(speedStr, "GBIT")
	} else if strings.HasSuffix(speedStr, "MBIT") {
		multiplier = 1
		numStr = strings.TrimSuffix(speedStr, "MBIT")
	} else {
		// No suffix, assume Mbps
		numStr = speedStr
	}

	numStr = strings.TrimSpace(numStr)

	speed, err := strconv.Atoi(numStr)
	if err != nil {
		return DefaultLinkSpeed
	}

	if speed <= 0 {
		return DefaultLinkSpeed
	}

	result := speed * multiplier
	if result <= 0 {
		return DefaultLinkSpeed
	}

	return result
}

// GetDeviceLinkSpeed retrieves the link speed of a network device and returns it in Mbps.
// Returns DefaultLinkSpeed if the device cannot be queried or speed is not available.
func GetDeviceLinkSpeed(deviceName string) int {
	// Import ubus here would create a dependency, so we'll call this from network-lan.go
	// This function serves as documentation of intent
	return DefaultLinkSpeed
}
