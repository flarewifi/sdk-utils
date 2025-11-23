//go:build !dev

package machineuid

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

var (
	mu         sync.Mutex
	machineUID string
	verified   bool
)

// GetMachineUID returns a unique identifier for the OpenWRT device.
// It uses:
// 1. CPU serial from /proc/cpuinfo (if available)
// 2. MAC addresses from all physical network interfaces (excludes virtual interfaces)
// 3. The combined identifiers are hashed using SHA-1
func GetMachineUID() string {
	mu.Lock()
	defer mu.Unlock()

	if machineUID != "" && verified {
		return machineUID
	}

	log.Println("[DEBUG] GetMachineUID: Starting machine UID generation")
	var identifiers []string

	// Get CPU serial if available
	serial := readCPUSerial()
	if serial != "" {
		log.Printf("[DEBUG] GetMachineUID: Found CPU serial: %s", serial)
		identifiers = append(identifiers, serial)
	} else {
		log.Println("[DEBUG] GetMachineUID: No CPU serial found")
	}

	// Get all physical network interface MACs (excludes docker, virbr, veth, etc.)
	allMACs := readAllNetworkMACs()
	log.Printf("[DEBUG] GetMachineUID: Found %d MAC addresses: %v", len(allMACs), allMACs)
	identifiers = append(identifiers, allMACs...)

	// If no identifiers at all, return empty string
	if len(identifiers) == 0 {
		log.Println("[DEBUG] GetMachineUID: No identifiers found, returning empty string")
		return ""
	}

	log.Printf("[DEBUG] GetMachineUID: Total identifiers: %v", identifiers)
	// Hash the combined identifiers
	uid := sdkutils.Sha1Hash(identifiers...)
	log.Printf("[DEBUG] GetMachineUID: Generated UID: %s", uid)

	if machineUID != "" && machineUID != uid {
		log.Printf("[DEBUG] GetMachineUID: Warning - Machine UID has changed from %s to %s", machineUID, uid)
	}

	// Verify if cached UID matches
	if machineUID != "" && machineUID == uid {
		verified = true
	}

	// Cache the result
	machineUID = uid

	return uid
}

// readInterfaceMAC reads the MAC address of a specific network interface
func readInterfaceMAC(iface string) string {
	log.Printf("[DEBUG] readInterfaceMAC: Reading MAC for interface: %s", iface)
	macPath := filepath.Join("/sys/class/net", iface, "address")
	data, err := os.ReadFile(macPath)
	if err != nil {
		log.Printf("[DEBUG] readInterfaceMAC: Failed to read MAC for %s: %v", iface, err)
		return ""
	}

	mac := strings.TrimSpace(string(data))
	log.Printf("[DEBUG] readInterfaceMAC: Raw MAC for %s: %s", iface, mac)

	// Validate MAC address (should not be empty, all zeros, or loopback)
	if mac == "" || mac == "00:00:00:00:00:00" || strings.HasPrefix(mac, "00:00:00") {
		log.Printf("[DEBUG] readInterfaceMAC: Invalid MAC for %s: %s", iface, mac)
		return ""
	}

	log.Printf("[DEBUG] readInterfaceMAC: Valid MAC for %s: %s", iface, mac)
	return mac
}

// readCPUSerial reads the serial number from /proc/cpuinfo (common on ARM devices)
func readCPUSerial() string {
	log.Println("[DEBUG] readCPUSerial: Reading CPU serial from /proc/cpuinfo")
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		log.Printf("[DEBUG] readCPUSerial: Failed to read /proc/cpuinfo: %v", err)
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "serial") {
			log.Printf("[DEBUG] readCPUSerial: Found serial line: %s", line)
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				serial := strings.TrimSpace(parts[1])
				log.Printf("[DEBUG] readCPUSerial: Extracted serial: %s", serial)
				if serial != "" && serial != "0" && serial != "0000000000000000" {
					log.Printf("[DEBUG] readCPUSerial: Valid serial found: %s", serial)
					return serial
				} else {
					log.Printf("[DEBUG] readCPUSerial: Invalid serial value: %s", serial)
				}
			}
		}
	}

	log.Println("[DEBUG] readCPUSerial: No valid CPU serial found")
	return ""
}

// readAllNetworkMACs reads MAC addresses from all available network interfaces
func readAllNetworkMACs() []string {
	log.Println("[DEBUG] readAllNetworkMACs: Reading network interfaces")
	netPath := "/sys/class/net"
	entries, err := os.ReadDir(netPath)
	if err != nil {
		log.Printf("[DEBUG] readAllNetworkMACs: Failed to read /sys/class/net: %v", err)
		return nil
	}

	log.Printf("[DEBUG] readAllNetworkMACs: Found %d network entries", len(entries))
	var macs []string
	for _, entry := range entries {
		ifaceName := entry.Name()
		log.Printf("[DEBUG] readAllNetworkMACs: Processing interface: %s", ifaceName)

		// Skip virtual and special interfaces
		if isVirtualInterface(ifaceName) {
			log.Printf("[DEBUG] readAllNetworkMACs: Skipping virtual/special interface: %s", ifaceName)
			continue
		}

		mac := readInterfaceMAC(ifaceName)
		if mac != "" {
			macs = append(macs, mac)
		}
	}

	// Sort for consistency
	sdkutils.TrimRedundantWords(strings.Join(macs, " "), " ")
	sort.Strings(macs)
	log.Printf("[DEBUG] readAllNetworkMACs: Final sorted MACs: %v", macs)
	return macs
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
