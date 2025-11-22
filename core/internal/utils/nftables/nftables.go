//go:build !dev

package nftables

import (
	"fmt"
	"log"
	"strings"
	"sync"

	jobque "tools/job-que"
	cmd "tools/shell"
)

const (
	internetTable   string = "internet"
	forward         string = "FORWARD"
	prerouting      string = "PREROUTING"
	connMacMap      string = "connected_macs_map"
	connIpMap       string = "connected_ips_map"
	connMacSet      string = "connected_macs_set"
	forwardPriority int    = 50 // Priority for internet FORWARD chain
)

var (
	nftMu         sync.RWMutex
	initCallbacks []func() error = []func() error{}
	nftQue                       = jobque.NewJobQue[any]()
)

func JumpChain(mac string) string {
	return "counter_" + strings.ReplaceAll(mac, ":", "")
}

func AddInitCallback(cb func() error) {
	nftMu.Lock()
	defer nftMu.Unlock()
	initCallbacks = append(initCallbacks, cb)
}

func Cleanup() {
	cmds := []string{
		fmt.Sprintf("nft flush map ip %s %s", internetTable, connIpMap),
		fmt.Sprintf("nft flush map ip %s %s", internetTable, connMacMap),
		fmt.Sprintf("nft flush set ip %s %s", internetTable, connMacSet),
		fmt.Sprintf("nft flush chain ip %s %s", internetTable, forward),
		fmt.Sprintf("nft flush chain ip %s %s", internetTable, prerouting),
		fmt.Sprintf("nft delete table ip %s", internetTable),
	}
	cmd.ExecAll(cmds)
}

func Setup() (err error) {
	Cleanup()

	cmds := []string{
		fmt.Sprintf("nft add table ip %s", internetTable),
		fmt.Sprintf("nft add chain ip %s %s '{ type nat hook prerouting priority dstnat; policy accept ; }'", internetTable, prerouting),
		fmt.Sprintf("nft add chain ip %s %s '{ type filter hook forward priority %d ; policy drop ; }'", internetTable, forward, forwardPriority),
		fmt.Sprintf("nft add map ip %s %s '{ type ipv4_addr : verdict ; counter; }'", internetTable, connIpMap),
		fmt.Sprintf("nft add map ip %s %s '{ type ether_addr : verdict ; counter; }'", internetTable, connMacMap),
		fmt.Sprintf("nft add set ip %s %s '{ type ether_addr; }'", internetTable, connMacSet),
		fmt.Sprintf("nft add rule %s %s ether saddr vmap @%s", internetTable, forward, connMacMap),
		fmt.Sprintf("nft add rule %s %s ip daddr vmap @%s", internetTable, forward, connIpMap),
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
			fmt.Sprintf("nft add rule ip %s %s ether saddr @%s counter accept", internetTable, prerouting, connMacSet),
			fmt.Sprintf("nft add rule ip %s %s iif %s tcp dport '{ 80, 443 }' counter dnat to %s", internetTable, prerouting, dev, routerIp),
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
	err := cmd.Exec(fmt.Sprintf("nft get element ip %s %s '{ %s }'", internetTable, connMacSet, mac), nil)
	return err == nil
}

func doConnect(ip string, mac string) error {
	cmds := []string{}
	connected := isConnected(mac)

	if !connected {
		cmds = []string{
			fmt.Sprintf("nft add element ip %s %s '{ %s : accept }'", internetTable, connIpMap, ip),
			fmt.Sprintf("nft add element ip %s %s '{ %s : accept }'", internetTable, connMacMap, mac),
			fmt.Sprintf("nft add element ip %s %s '{ %s }'", internetTable, connMacSet, mac),
		}

		return cmd.ExecAll(cmds)
	}

	return nil
}

func doDisconnect(ip string, mac string) error {
	connected := isConnected(mac)
	if connected {
		cmds := []string{
			fmt.Sprintf("nft delete element ip %s %s '{ %s : accept }'", internetTable, connIpMap, ip),
			fmt.Sprintf("nft delete element ip %s %s { %s : accept }", internetTable, connMacMap, mac),
			fmt.Sprintf("nft delete element ip %s %s '{ %s }'", internetTable, connMacSet, mac),
		}
		return cmd.ExecAll(cmds)
	}
	return nil
}
