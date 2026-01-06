//go:build !dev

package machineuid

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	machineIDCacheFile = "/etc/.mid"
	maxRetries         = 10
	retryDelay         = 3 * time.Second
)

var (
	mu         sync.Mutex
	machineUID string
	verified   bool
)

// readCachedMachineID reads the cached machine ID from /etc/.mid
func readCachedMachineID() string {
	data, err := os.ReadFile(machineIDCacheFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// writeCachedMachineID writes the machine ID to /etc/.mid
func writeCachedMachineID(uid string) {
	err := os.WriteFile(machineIDCacheFile, []byte(uid), 0644)
	if err != nil {
		log.Printf("Warning: Failed to write machine ID cache to %s: %v", machineIDCacheFile, err)
	}
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

	// Hash the combined identifiers
	return strings.ToUpper(sdkutils.Sha1Hash(identifiers...))
}

// GetMachineUID returns a unique identifier for the OpenWRT device.
// It uses:
// 1. CPU serial from /proc/cpuinfo (if available)
// 2. MAC addresses from accepted physical network interfaces (wan, lan, lan1-9, eth0-9, eth*, lan*)
// 3. The combined identifiers are hashed using SHA-1
// 4. Caches the result to /etc/.mid for consistency
//
// Returns (oldMachineID, newMachineID):
// - If machine ID hasn't changed: returns ("", currentID)
// - If machine ID changed: returns (cachedID, newID)
func GetMachineUID() (string, string) {
	mu.Lock()
	defer mu.Unlock()

	if machineUID != "" && verified {
		return "", machineUID
	}

	// Read cached machine ID
	cachedID := readCachedMachineID()

	// Calculate current machine ID
	currentID := calculateMachineUID()
	if currentID == "" {
		return "", ""
	}

	// If no cached ID exists, this is first run
	if cachedID == "" {
		machineUID = currentID
		verified = true
		writeCachedMachineID(currentID)
		return "", currentID
	}

	// If cached ID matches current ID, we're done
	if cachedID == currentID {
		machineUID = currentID
		verified = true
		return "", currentID
	}

	// Machine ID mismatch - retry with delays
	log.Printf("Machine ID mismatch: cached=%s, current=%s. Retrying...", cachedID, currentID)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		time.Sleep(retryDelay)

		retryID := calculateMachineUID()
		if retryID == cachedID {
			log.Printf("Machine ID matched cached value after %d attempts", attempt)
			machineUID = cachedID
			verified = true
			return "", cachedID
		}

		log.Printf("Retry %d/%d: Machine ID still %s (expected %s)", attempt, maxRetries, retryID, cachedID)
		currentID = retryID
	}

	// After all retries, machine ID has changed
	log.Printf("Machine ID changed after %d retries: %s -> %s", maxRetries, cachedID, currentID)

	// Update cache with new ID
	machineUID = currentID
	verified = true
	writeCachedMachineID(currentID)

	return cachedID, currentID
}

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
