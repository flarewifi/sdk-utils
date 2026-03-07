//go:build dev

package nftables

import (
	"fmt"
	"log"
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
	connMacSet       string = "connected_macs_set"
)

type connectedClient struct {
	ip  string
	mac string
}

var (
	nftMu         sync.RWMutex
	initCallbacks []func() error = []func() error{}
	nftQue                       = jobque.NewJobQueue[any]()
	connTable                    = map[string]bool{}
	connClients                  = make(map[string]*connectedClient) // mac -> client info
)

func AddInitCallback(cb func() error) {
	nftMu.Lock()
	defer nftMu.Unlock()
	initCallbacks = append(initCallbacks, cb)
}

func Cleanup() {}

func Setup() (err error) {
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

	connected := isConnected(normalizedMAC)

	if !connected {
		connTable[normalizedMAC] = true
		connClients[normalizedMAC] = &connectedClient{
			ip:  ip,
			mac: normalizedMAC,
		}
	}

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

	connected := isConnected(normalizedMAC)
	if connected {
		delete(connTable, normalizedMAC)
		delete(connClients, normalizedMAC)
	}
	log.Println("nftables disconnected: " + normalizedMAC)
	return nil
}
