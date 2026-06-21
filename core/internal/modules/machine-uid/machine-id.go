//go:build !dev

package machineuid

import (
	"crypto/sha1"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	sdkutils "github.com/flarewifi/sdk-utils"
)

const (
	machineIDCacheFile = "/etc/.mid"

	// readableAlphabet excludes ambiguous chars: 0/O, 1/I/l
	readableAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

var (
	mu         sync.Mutex
	machineUID string
)

// readCachedMachineID reads the cached machine ID from /etc/.mid
func readCachedMachineID() string {
	data, err := os.ReadFile(machineIDCacheFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// WriteCachedMachineID writes the machine ID to /etc/.mid
// Called after successful cloud sync when machine ID changes
func WriteCachedMachineID(uid string) {
	mu.Lock()
	defer mu.Unlock()

	err := os.WriteFile(machineIDCacheFile, []byte(uid), 0644)
	if err != nil {
		return
	}

	// Update in-memory state
	machineUID = uid
}

// calculateMachineUID calculates the machine UID from system identifiers
func calculateMachineUID() string {
	var identifiers []string

	// Get CPU serial if available
	serial := readCPUSerial()
	if serial != "" {
		identifiers = append(identifiers, serial)
	}

	// Get all physical network interface MACs (excludes docker, virbr, veth, etc.)
	allMACs := readAllNetworkMACs()
	identifiers = append(identifiers, allMACs...)

	// If no identifiers at all, return empty string
	if len(identifiers) == 0 {
		return ""
	}

	// Hash the combined identifiers and encode to readable 16-char string
	hash := sha1.Sum([]byte(strings.Join(identifiers, "")))
	return encodeReadable(hash[:], 16)
}

// GetMachineUIDWithChange detects if the machine ID has changed.
// Returns (cachedID, calculatedID):
// - If no cached ID (new machine): returns ("", newID)
// - If cached == calculated: returns ("", cachedID) - no change
// - If cached != calculated: returns (cachedID, newID) - machine ID changed!
//
// This function does NOT update the cache - caller must handle that after cloud sync.
func GetMachineUIDWithChange() (string, string) {
	mu.Lock()
	defer mu.Unlock()

	// Read cached machine ID from file
	cachedID := readCachedMachineID()

	// Calculate current machine ID from hardware
	calculatedID := calculateMachineUID()
	if calculatedID == "" {
		return "", ""
	}

	// No cached ID - this is a new machine
	if cachedID == "" {
		return "", calculatedID
	}

	// Check if IDs match
	if cachedID == calculatedID {
		return "", cachedID
	}

	// Machine ID has changed
	return cachedID, calculatedID
}

// GetMachineUID returns the current machine ID.
// If already resolved, returns cached value.
// Otherwise reads from /etc/.mid or calculates for new machines.
//
// Returns ("", machineID) - first value is always empty for backward compatibility.
func GetMachineUID() (string, string) {
	mu.Lock()
	defer mu.Unlock()

	// Return cached in-memory value if available
	if machineUID != "" {
		return "", machineUID
	}

	// Read cached machine ID from file
	cachedID := readCachedMachineID()

	// If cached ID exists, use it
	if cachedID != "" {
		machineUID = cachedID
		return "", cachedID
	}

	// No cached ID - this is a new machine
	// Calculate and cache the new 16-char format ID
	newID := calculateMachineUID()
	if newID == "" {
		return "", ""
	}

	// Write to cache file
	os.WriteFile(machineIDCacheFile, []byte(newID), 0644)

	machineUID = newID
	return "", newID
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// readInterfaceMAC reads the MAC address of a specific network interface
func readInterfaceMAC(iface string) string {
	macPath := filepath.Join("/sys/class/net", iface, "address")
	data, err := os.ReadFile(macPath)
	if err != nil {
		return ""
	}

	mac := strings.TrimSpace(string(data))

	// Validate MAC address (should not be empty, all zeros, or loopback)
	if mac == "" || mac == "00:00:00:00:00:00" || strings.HasPrefix(mac, "00:00:00") {
		return ""
	}

	return mac
}

// readCPUSerial reads the serial number from /proc/cpuinfo (common on ARM devices)
func readCPUSerial() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "serial") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				serial := strings.TrimSpace(parts[1])
				if serial != "" && serial != "0" && serial != "0000000000000000" {
					return serial
				}
			}
		}
	}

	return ""
}

// isAcceptedInterface checks if an interface matches accepted patterns
// Accepts: wan, lan, lan1-9, eth0-9, eth*, lan*
func isAcceptedInterface(ifaceName string) bool {
	// Exact matches
	if ifaceName == "wan" || ifaceName == "lan" {
		return true
	}

	// lan1-9 (exactly 4 characters: lan + single digit 1-9)
	if strings.HasPrefix(ifaceName, "lan") && len(ifaceName) == 4 {
		digit := ifaceName[3]
		if digit >= '1' && digit <= '9' {
			return true
		}
	}

	// eth0-9 (exactly 4 characters: eth + single digit 0-9)
	if strings.HasPrefix(ifaceName, "eth") && len(ifaceName) == 4 {
		digit := ifaceName[3]
		if digit >= '0' && digit <= '9' {
			return true
		}
	}

	// eth* (any interface starting with eth)
	if strings.HasPrefix(ifaceName, "eth") {
		return true
	}

	// lan* (any interface starting with lan)
	if strings.HasPrefix(ifaceName, "lan") {
		return true
	}

	return false
}

// readAllNetworkMACs reads MAC addresses from all available network interfaces
func readAllNetworkMACs() []string {
	netPath := "/sys/class/net"
	entries, err := os.ReadDir(netPath)
	if err != nil {
		return nil
	}

	var macs []string
	for _, entry := range entries {
		ifaceName := entry.Name()

		// Skip virtual and special interfaces
		if isVirtualInterface(ifaceName) {
			continue
		}

		// Only accept specific interface patterns
		if !isAcceptedInterface(ifaceName) {
			continue
		}

		mac := readInterfaceMAC(ifaceName)
		if mac != "" {
			macs = append(macs, mac)
		}
	}

	// Sort for consistency and group duplicates
	sort.Strings(macs)
	uniqueMACs := sdkutils.TrimRedundantWords(strings.Join(macs, " "), " ")
	return strings.Fields(uniqueMACs)
}

// isVirtualInterface checks if an interface is virtual (should be excluded)
func isVirtualInterface(ifaceName string) bool {
	// Skip loopback
	if ifaceName == "lo" {
		return true
	}

	// Skip common virtual interface prefixes
	virtualPrefixes := []string{
		"ifb",    // Intermediate Functional Block devices
		"vlan",   // VLAN interfaces
		"vnet",   // Virtual network interfaces (KVM/QEMU)
		"veth",   // Virtual Ethernet devices
		"br-",    // Bridge interfaces
		"docker", // Docker interfaces
		"virbr",  // Virtual bridge (libvirt)
		"tun",    // TUN/TAP devices
		"tap",    // TAP devices
		"wg",     // WireGuard interfaces
		"ppp",    // PPP interfaces
		"sit",    // IPv6-in-IPv4 tunnels
		"gre",    // GRE tunnels
		"ip6tnl", // IPv6 tunnels
	}

	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(ifaceName, prefix) {
			return true
		}
	}

	// Check for VLAN notation (e.g., eth0.100)
	if strings.Contains(ifaceName, ".") {
		return true
	}

	return false
}

// encodeReadable encodes bytes to a readable string using the readable alphabet
// Uses all 20 bytes of SHA-1 hash to generate 16 unique characters
func encodeReadable(data []byte, length int) string {
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		// Use different byte combinations for each position to avoid repetition
		// Mix bytes from different parts of the hash
		byteIdx := i % len(data)
		nextIdx := (i + 7) % len(data)   // Prime offset for better mixing
		thirdIdx := (i + 13) % len(data) // Another prime offset
		combined := int(data[byteIdx]) + int(data[nextIdx])*256 + int(data[thirdIdx])*17
		result[i] = readableAlphabet[combined%len(readableAlphabet)]
	}
	return string(result)
}
