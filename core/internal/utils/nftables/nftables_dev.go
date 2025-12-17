//go:build dev

package nftables

import (
	"fmt"
	"log"
	"sync"
	"time"

	jobque "tools/job-que"
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

var nftQue = jobque.NewJobQue[bool]()
var nftMu sync.RWMutex
var initCallbacks []func() error = []func() error{}
var connTable map[string]bool

func init() {
	connTable = map[string]bool{}
}

func runInitCallbacks() {
	// for _, cb := range initCallbacks {
	// err := cb()
	// if err != nil {
	// log.Println(err)
	// }
	// }
}

func isConnected(mac string) bool {
	_, ok := connTable[mac]
	return ok
}

func doConnect(_ string, mac string) error {
	connected := isConnected(mac)

	if !connected {
		connTable[mac] = true
	}

	log.Println("nftables connected: " + mac)
	return nil
}

func doDisconnect(_ string, mac string) error {
	connected := isConnected(mac)
	if connected {
		delete(connTable, mac)
	}
	log.Println("nftables disconnected: " + mac)
	return nil
}

func Cleanup() {}

func Setup() (err error) {
	nftMu.Lock()
	defer nftMu.Unlock()

	Cleanup()
	runInitCallbacks()
	return nil
}

func SetupCaptivePortal(dev string, routerIp string) (err error) {
	contextInfo := fmt.Sprintf("Device=%s, RouterIP=%s", dev, routerIp)

	_, err = nftQue.ExecWithTimeout(
		4*time.Second,
		"Setup Captive Portal",
		contextInfo,
		func() (bool, error) {
			return true, nil
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
		func() (bool, error) {
			nftMu.Lock()
			defer nftMu.Unlock()
			err := doConnect(ip, mac)
			return true, err
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
		func() (bool, error) {
			nftMu.Lock()
			defer nftMu.Unlock()
			err := doDisconnect(ip, mac)
			return true, err
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
		func() (bool, error) {
			nftMu.RLock()
			defer nftMu.RUnlock()
			return isConnected(mac), nil
		},
	)

	if err != nil {
		log.Printf("[ERROR] IsConnected check failed for %s: %v", mac, err)
		return false
	}

	return result
}

func AddInitCallback(cb func() error) {
	nftMu.Lock()
	defer nftMu.Unlock()
	initCallbacks = append(initCallbacks, cb)
}
