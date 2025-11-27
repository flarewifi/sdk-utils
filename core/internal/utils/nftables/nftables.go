//go:build !dev

package nftables

import (
	"fmt"
	"log"
	"sync"

	jobque "tools/job-que"
	cmd "tools/shell"
)

const (
	fw4Table        string = "fw4"      // OpenWrt default table (for jump rules)
	internetTable   string = "internet" // Our custom table
	tableFamily     string = "inet"     // inet family (handles both ipv4 and ipv6)
	forwardChain    string = "forward"
	preroutingChain string = "prerouting"
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
	_, err = nftQue.Exec(func() (any, error) {
		cmds := []string{
			// Add rules to our custom prerouting chain
			// Allow already authenticated devices to bypass captive portal
			fmt.Sprintf("nft add rule %s %s %s ether saddr @%s counter accept", tableFamily, internetTable, preroutingChain, connMacSet),
			// Redirect HTTP/HTTPS traffic to captive portal
			fmt.Sprintf("nft add rule %s %s %s iif %s tcp dport '{ 80, 443 }' counter dnat ip to %s", tableFamily, internetTable, preroutingChain, dev, routerIp),
		}
		err := cmd.ExecAll(cmds)
		return nil, err
	})
	return err
}

func Connect(ip string, mac string) error {
	_, err := nftQue.Exec(func() (any, error) {
		err := doConnect(ip, mac)
		return nil, err
	})
	return err
}

func Disconnect(ip string, mac string) error {
	_, err := nftQue.Exec(func() (any, error) {
		err := doDisconnect(ip, mac)
		return nil, err
	})
	return err
}

func IsConnected(mac string) bool {
	result, _ := nftQue.Exec(func() (any, error) {
		return isConnected(mac), nil
	})
	return result.(bool)
}

func runInitCallbacks() {
	nftMu.RLock()
	defer nftMu.RUnlock()
	for _, cb := range initCallbacks {
		err := cb()
		if err != nil {
			log.Println(err)
		}
	}
}

func isConnected(mac string) bool {
	err := cmd.Exec(fmt.Sprintf("nft get element %s %s %s '{ %s }'", tableFamily, internetTable, connMacSet, mac), nil)
	return err == nil
}

func doConnect(ip string, mac string) error {
	cmds := []string{}
	connected := isConnected(mac)

	if !connected {
		cmds = []string{
			fmt.Sprintf("nft add element %s %s %s '{ %s : accept }'", tableFamily, internetTable, connIpMap, ip),
			fmt.Sprintf("nft add element %s %s %s '{ %s : accept }'", tableFamily, internetTable, connMacMap, mac),
			fmt.Sprintf("nft add element %s %s %s '{ %s }'", tableFamily, internetTable, connMacSet, mac),
		}

		return cmd.ExecAll(cmds)
	}

	return nil
}

func doDisconnect(ip string, mac string) error {
	connected := isConnected(mac)
	if connected {
		cmds := []string{
			fmt.Sprintf("nft delete element %s %s %s '{ %s : accept }'", tableFamily, internetTable, connIpMap, ip),
			fmt.Sprintf("nft delete element %s %s %s { %s : accept }", tableFamily, internetTable, connMacMap, mac),
			fmt.Sprintf("nft delete element %s %s %s '{ %s }'", tableFamily, internetTable, connMacSet, mac),
		}
		return cmd.ExecAll(cmds)
	}
	return nil
}
