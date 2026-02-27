//go:build !dev

package nftables

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	jobque "core/utils/job-que"
	cmd "core/utils/shell"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	internetTable   string = "internet" // Our custom table
	tableFamily     string = "inet"     // inet family (handles both ipv4 and ipv6)
	forwardChain     string = "forward"
	preroutingChain  string = "prerouting"
	postroutingChain string = "postrouting"
	connMacMap      string = "connected_macs_map"
	connIpMap       string = "connected_ips_map"
	connMacSet      string = "connected_macs_set"
)

var (
	nftMu         sync.RWMutex
	initCallbacks []func() error = []func() error{}
	nftQue                       = jobque.NewJobQue[any]()
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

	cmds := []string{
		// Create our custom internet table
		fmt.Sprintf("nft add table %s %s", tableFamily, internetTable),

		// Create custom forward and prerouting chains as base chains with hooks (priority -1 runs before fw4)
		fmt.Sprintf("nft add chain %s %s %s '{ type filter hook forward priority -250; policy drop; }'", tableFamily, internetTable, forwardChain),
		fmt.Sprintf("nft add chain %s %s %s '{ type nat hook prerouting priority -1; policy accept; }'", tableFamily, internetTable, preroutingChain),

		// Create maps and sets in our custom table
		fmt.Sprintf("nft add map %s %s %s '{ type ipv4_addr : verdict ; counter; }'", tableFamily, internetTable, connIpMap),
		fmt.Sprintf("nft add map %s %s %s '{ type ether_addr : verdict ; counter; }'", tableFamily, internetTable, connMacMap),
		fmt.Sprintf("nft add set %s %s %s '{ type ether_addr; }'", tableFamily, internetTable, connMacSet),

		// Create postrouting chain for anti-tethering (TTL set)
		// Sets outgoing TTL to 1 so tethered devices cannot forward packets (TTL drops to 0)
		fmt.Sprintf("nft add chain %s %s %s '{ type filter hook postrouting priority 0; policy accept; }'", tableFamily, internetTable, postroutingChain),

		// Add rules to our custom forward chain
		// Verdict maps will accept if MAC/IP is in the map, otherwise continue to drop rule
		fmt.Sprintf("nft add rule %s %s %s ether saddr vmap @%s", tableFamily, internetTable, forwardChain, connMacMap),
		fmt.Sprintf("nft add rule %s %s %s ip daddr vmap @%s", tableFamily, internetTable, forwardChain, connIpMap),
	}

	err = cmd.ExecAll(cmds)
	if err != nil {
		return err
	}

	runInitCallbacks()
	return nil
}

func SetupCaptivePortal(dev string, routerIp string) (err error) {
	contextInfo := fmt.Sprintf("Device=%s, RouterIP=%s", dev, routerIp)

	_, err = nftQue.ExecWithTimeout(
		4*time.Second,
		"Setup Captive Portal",
		contextInfo,
		func() (any, error) {
			cmds := []string{
				// Add rules to our custom prerouting chain
				// Allow already authenticated devices to bypass captive portal
				fmt.Sprintf("nft add rule %s %s %s ether saddr @%s counter accept", tableFamily, internetTable, preroutingChain, connMacSet),
				// Redirect HTTP/HTTPS traffic to captive portal
				fmt.Sprintf("nft add rule %s %s %s iif %s tcp dport '{ 80, 443 }' counter dnat ip to %s", tableFamily, internetTable, preroutingChain, dev, routerIp),

				// Anti-tethering: set TTL=1 on packets going out through this LAN device
				// Direct clients receive TTL=1 and process it normally
				// Tethered devices try to forward, TTL drops to 0, packet is dropped
				fmt.Sprintf("nft add rule %s %s %s oifname %s ip ttl set 1", tableFamily, internetTable, postroutingChain, dev),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			err := cmd.ExecAllWithContext(ctx, cmds)
			return nil, err
		},
	)
	return err
}

func Connect(ip string, mac string) error {
	contextInfo := fmt.Sprintf("IP=%s, MAC=%s", ip, mac)

	_, err := nftQue.ExecWithTimeout(
		3*time.Second,
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
		3*time.Second,
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
		1*time.Second,
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

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err = cmd.ExecWithContext(ctx, fmt.Sprintf("nft get element %s %s %s '{ %s }'", tableFamily, internetTable, connMacSet, normalizedMAC), nil)
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

	cmds := []string{}
	connected := isConnected(normalizedMAC)

	if !connected {
		cmds = []string{
			fmt.Sprintf("nft add element %s %s %s '{ %s : accept }'", tableFamily, internetTable, connIpMap, ip),
			fmt.Sprintf("nft add element %s %s %s '{ %s : accept }'", tableFamily, internetTable, connMacMap, normalizedMAC),
			fmt.Sprintf("nft add element %s %s %s '{ %s }'", tableFamily, internetTable, connMacSet, normalizedMAC),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		return cmd.ExecAllWithContext(ctx, cmds)
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
		cmds := []string{
			fmt.Sprintf("nft delete element %s %s %s '{ %s : accept }'", tableFamily, internetTable, connIpMap, ip),
			fmt.Sprintf("nft delete element %s %s %s '{ %s : accept }'", tableFamily, internetTable, connMacMap, normalizedMAC),
			fmt.Sprintf("nft delete element %s %s %s '{ %s }'", tableFamily, internetTable, connMacSet, normalizedMAC),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		return cmd.ExecAllWithContext(ctx, cmds)
	}
	return nil
}
