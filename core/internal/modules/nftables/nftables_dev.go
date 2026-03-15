//go:build dev

package nftables

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	jobque "core/utils/job-que"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	internetTable    string = "internet" // Our custom table
	tableFamily      string = "inet"     // inet family (handles both ipv4 and ipv6)
	forwardChain     string = "forward"
	preroutingChain  string = "prerouting"
	postroutingChain string = "postrouting"
	connMacMap       string = "connected_macs_map"
	connIpMap        string = "connected_ips_map"
	connIp6Map       string = "connected_ips6_map"
	connMacSet       string = "connected_macs_set"
)

var (
	nftMu  sync.RWMutex
	nftQue = jobque.NewJobQueue[any]()
	// connTable tracks connected MACs (MAC → true).
	connTable = map[string]bool{}

	// ipToMac maps connected IP address (IPv4 or IPv6) → uppercase normalized MAC.
	// Populated on Connect, evicted on Disconnect.
	// Guarded by nftMu.
	ipToMac = make(map[string]string)

	// macToIps maps connected MAC → set of IPs registered for that device.
	// Guarded by nftMu.
	macToIps = make(map[string]map[string]bool)
)

func Cleanup() {}

func Setup() (err error) {
	Cleanup()
	return nil
}

// SetupCaptivePortal installs prerouting DNAT rules for a LAN interface.
// routerIp4 is the LAN-facing IPv4 address; routerIp6 is the LAN-facing IPv6
// address (may be empty if the interface has no IPv6 address yet).
func SetupCaptivePortal(dev string, routerIp4 string, routerIp6 string) (err error) {
	contextInfo := fmt.Sprintf("Device=%s, RouterIPv4=%s, RouterIPv6=%s", dev, routerIp4, routerIp6)

	_, err = nftQue.ExecWithTimeout(
		4*time.Second,
		"Setup Captive Portal",
		contextInfo,
		func() (any, error) {
			return nil, nil
		},
	)
	return err
}

func Connect(ip string, mac string) error {
	contextInfo := fmt.Sprintf("IP=%s, MAC=%s", ip, mac)

	_, err := nftQue.ExecWithTimeout(
		30*time.Second,
		"Connect Device",
		contextInfo,
		func() (any, error) {
			err := doConnect(ip, mac)
			return nil, err
		},
	)
	return err
}

func Disconnect(ip string, mac string) error {
	contextInfo := fmt.Sprintf("IP=%s, MAC=%s", ip, mac)

	_, err := nftQue.ExecWithTimeout(
		30*time.Second,
		"Disconnect Device",
		contextInfo,
		func() (any, error) {
			err := doDisconnect(ip, mac)
			return nil, err
		},
	)
	return err
}

func IsConnected(mac string) bool {
	contextInfo := fmt.Sprintf("MAC=%s", mac)

	result, err := nftQue.ExecWithTimeout(
		30*time.Second,
		"Check Connection Status",
		contextInfo,
		func() (any, error) {
			return isConnected(mac), nil
		},
	)

	if err != nil {
		log.Printf("[ERROR] IsConnected check failed for %s: %v", mac, err)
		return false
	}

	return result.(bool)
}

// GetMacByIp returns the normalized uppercase MAC address for a currently
// connected IP address (IPv4 or IPv6), or an empty string if the IP is not
// in the cache.
func GetMacByIp(ip string) string {
	nftMu.RLock()
	defer nftMu.RUnlock()
	return ipToMac[ip]
}

// GetMacsByIps returns a map of IP→MAC for a batch of IP addresses, acquiring
// the lock only once. This is more efficient than calling GetMacByIp in a loop.
// IPs not found in the cache are omitted from the result map.
func GetMacsByIps(ips []string) map[string]string {
	nftMu.RLock()
	defer nftMu.RUnlock()

	result := make(map[string]string, len(ips))
	for _, ip := range ips {
		if mac := ipToMac[ip]; mac != "" {
			result[ip] = mac
		}
	}
	return result
}

// isIPv6 returns true if ip is a valid IPv6 address (not IPv4 or IPv4-mapped).
func isIPv6(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.To4() == nil
}

func isConnected(mac string) bool {
	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		log.Printf("[ERROR] isConnected: invalid MAC address '%s': %v", mac, err)
		return false
	}

	_, ok := connTable[normalizedMAC]
	return ok
}

func doConnect(ip string, mac string) error {
	// Validate IP address
	if _, err := sdkutils.ValidateIPAddress(ip); err != nil {
		return fmt.Errorf("invalid IP address: %v", err)
	}

	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %v", err)
	}

	// Mark MAC as connected (idempotent for dual-stack second call).
	connTable[normalizedMAC] = true

	// Record IP→MAC and MAC→IPs mappings for traffic accounting and GetMacByIp().
	nftMu.Lock()
	ipToMac[ip] = normalizedMAC
	if macToIps[normalizedMAC] == nil {
		macToIps[normalizedMAC] = make(map[string]bool)
	}
	macToIps[normalizedMAC][ip] = true
	nftMu.Unlock()

	log.Println("nftables connected: " + normalizedMAC + " (" + ip + ")")
	return nil
}

func doDisconnect(ip string, mac string) error {
	// Validate IP address
	if _, err := sdkutils.ValidateIPAddress(ip); err != nil {
		return fmt.Errorf("invalid IP address: %v", err)
	}

	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %v", err)
	}

	// Remove this IP from both lookup maps.  Only remove the MAC from connTable
	// when all its IPs have been disconnected (handles dual-stack correctly).
	nftMu.Lock()
	delete(ipToMac, ip)
	remainingIPs := 0
	if ips, ok := macToIps[normalizedMAC]; ok {
		delete(ips, ip)
		remainingIPs = len(ips)
		if remainingIPs == 0 {
			delete(macToIps, normalizedMAC)
		}
	}
	nftMu.Unlock()

	if remainingIPs == 0 {
		delete(connTable, normalizedMAC)
	}

	log.Println("nftables disconnected IP: " + ip + " (" + normalizedMAC + ")")
	return nil
}
