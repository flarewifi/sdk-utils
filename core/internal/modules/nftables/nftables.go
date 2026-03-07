//go:build !dev

package nftables

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	jobque "core/utils/job-que"
	cmd "core/utils/shell"

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
	connMacSet       string = "connected_macs_set"
)

var (
	nftMu         sync.RWMutex
	initCallbacks []func() error = []func() error{}
	nftQue                       = jobque.NewJobQueue[any]()
)

func AddInitCallback(cb func() error) {
	nftMu.Lock()
	defer nftMu.Unlock()
	initCallbacks = append(initCallbacks, cb)
}

func Cleanup() {
	cmds := []string{
		// Delete our custom table (this removes all chains, maps, sets, and rules within it)
		fmt.Sprintf("nft delete table %s %s 2>/dev/null || true", tableFamily, internetTable),
	}
	cmd.ExecAll(cmds)
}

func Setup() (err error) {
	Cleanup()

	// Build nft batch script for atomic execution
	var batch strings.Builder

	// Create our custom internet table
	batch.WriteString(fmt.Sprintf("add table %s %s\n", tableFamily, internetTable))

	// Create custom forward and prerouting chains as base chains with hooks (priority -1 runs before fw4)
	batch.WriteString(fmt.Sprintf("add chain %s %s %s { type filter hook forward priority -250; policy drop; }\n", tableFamily, internetTable, forwardChain))
	batch.WriteString(fmt.Sprintf("add chain %s %s %s { type nat hook prerouting priority -1; policy accept; }\n", tableFamily, internetTable, preroutingChain))

	// Create maps and sets in our custom table
	batch.WriteString(fmt.Sprintf("add map %s %s %s { type ipv4_addr : verdict ; counter; }\n", tableFamily, internetTable, connIpMap))
	batch.WriteString(fmt.Sprintf("add map %s %s %s { type ether_addr : verdict ; counter; }\n", tableFamily, internetTable, connMacMap))
	batch.WriteString(fmt.Sprintf("add set %s %s %s { type ether_addr; }\n", tableFamily, internetTable, connMacSet))

	// Create postrouting chain for anti-tethering (TTL set)
	// Sets outgoing TTL to 1 so tethered devices cannot forward packets (TTL drops to 0)
	batch.WriteString(fmt.Sprintf("add chain %s %s %s { type filter hook postrouting priority 0; policy accept; }\n", tableFamily, internetTable, postroutingChain))

	// Add rules to our custom forward chain
	// Verdict maps will accept if MAC/IP is in the map, otherwise continue to drop rule
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ether saddr vmap @%s\n", tableFamily, internetTable, forwardChain, connMacMap))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ip daddr vmap @%s\n", tableFamily, internetTable, forwardChain, connIpMap))

	// Execute batch command using nft -f - with heredoc for atomic execution
	nftCmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
	err = cmd.Exec(nftCmd, nil)
	if err != nil {
		return err
	}

	runInitCallbacks()
	return nil
}

func SetupCaptivePortal(dev string, routerIp string) (err error) {
	contextInfo := fmt.Sprintf("Device=%s, RouterIP=%s", dev, routerIp)

	_, err = nftQue.ExecWithTimeout(
		30*time.Second,
		"Setup Captive Portal",
		contextInfo,
		func() (any, error) {
			// Build nft batch script for atomic execution
			var batch strings.Builder

			// Add rules to our custom prerouting chain
			// Allow already authenticated devices to bypass captive portal
			batch.WriteString(fmt.Sprintf("add rule %s %s %s ether saddr @%s counter accept\n", tableFamily, internetTable, preroutingChain, connMacSet))
			// Redirect HTTP/HTTPS traffic to captive portal
			batch.WriteString(fmt.Sprintf("add rule %s %s %s iif %s tcp dport { 80, 443 } counter dnat ip to %s\n", tableFamily, internetTable, preroutingChain, dev, routerIp))

			// Anti-tethering: set TTL=1 on packets going out through this LAN device
			// Direct clients receive TTL=1 and process it normally
			// Tethered devices try to forward, TTL drops to 0, packet is dropped
			batch.WriteString(fmt.Sprintf("add rule %s %s %s oifname %s ip ttl set 1\n", tableFamily, internetTable, postroutingChain, dev))

			// Execute batch command using nft -f - with heredoc for atomic execution
			nftCmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
			return nil, cmd.Exec(nftCmd, nil)
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

func runInitCallbacks() {
	// Copy the callbacks slice while holding the lock, then release
	// before executing. This prevents deadlock if a callback tries
	// to call AddInitCallback.
	nftMu.RLock()
	callbacks := make([]func() error, len(initCallbacks))
	copy(callbacks, initCallbacks)
	nftMu.RUnlock()

	for _, cb := range callbacks {
		err := cb()
		if err != nil {
			log.Println(err)
		}
	}
}

func isConnected(mac string) bool {
	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		log.Printf("[ERROR] isConnected: invalid MAC address '%s': %v", mac, err)
		return false
	}

	err = cmd.Exec(fmt.Sprintf("nft get element %s %s %s '{ %s }'", tableFamily, internetTable, connMacSet, normalizedMAC), nil)
	return err == nil
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

	connected := isConnected(normalizedMAC)

	if !connected {
		// Build nft batch script for atomic execution
		var batch strings.Builder
		batch.WriteString(fmt.Sprintf("add element %s %s %s { %s : accept }\n", tableFamily, internetTable, connIpMap, ip))
		batch.WriteString(fmt.Sprintf("add element %s %s %s { %s : accept }\n", tableFamily, internetTable, connMacMap, normalizedMAC))
		batch.WriteString(fmt.Sprintf("add element %s %s %s { %s }\n", tableFamily, internetTable, connMacSet, normalizedMAC))

		// Execute batch command using nft -f - with heredoc for atomic execution
		nftCmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
		return cmd.Exec(nftCmd, nil)
	}

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

	connected := isConnected(normalizedMAC)
	if connected {
		// Build nft batch script for atomic execution
		var batch strings.Builder
		batch.WriteString(fmt.Sprintf("delete element %s %s %s { %s : accept }\n", tableFamily, internetTable, connIpMap, ip))
		batch.WriteString(fmt.Sprintf("delete element %s %s %s { %s : accept }\n", tableFamily, internetTable, connMacMap, normalizedMAC))
		batch.WriteString(fmt.Sprintf("delete element %s %s %s { %s }\n", tableFamily, internetTable, connMacSet, normalizedMAC))

		// Execute batch command using nft -f - with heredoc for atomic execution
		nftCmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
		return cmd.Exec(nftCmd, nil)
	}
	return nil
}
