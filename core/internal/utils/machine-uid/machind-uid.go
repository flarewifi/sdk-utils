//go:build !dev

package machineuid

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// GetMachineUID returns a unique identifier for the OpenWRT device.
// It uses:
// 1. CPU serial from /proc/cpuinfo (if available)
// 2. MAC addresses from all physical network interfaces (excludes virtual interfaces)
// 3. The combined identifiers are hashed using SHA-1
func GetMachineUID() string {
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
	return sdkutils.Sha1Hash(identifiers...)
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

// readAllNetworkMACs reads MAC addresses from all available network interfaces
func readAllNetworkMACs() []string {
	netPath := "/sys/class/net"
	entries, err := os.ReadDir(netPath)
	if err != nil {
		return nil
	}

	var macs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		ifaceName := entry.Name()

		// Skip virtual and special interfaces
		if isVirtualInterface(ifaceName) {
			continue
		}

		mac := readInterfaceMAC(ifaceName)
		if mac != "" {
			macs = append(macs, mac)
		}
	}

	// Sort for consistency
	sort.Strings(macs)

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
