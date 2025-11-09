//go:build dev

package nftables

import (
	"log"
	"strings"
	"sync"

	jobque "tools/job-que"
)

const (
	// internetTable string = "internet"
	// forward       string = "FORWARD"
	// prerouting    string = "PREROUTING"
	connMacMap string = "connected_macs_map"
	connIpMap  string = "connected_ips_map"
	// connMacSet    string = "connected_macs_set"
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

func JumpChain(mac string) string {
	return "counter_" + strings.ReplaceAll(mac, ":", "")
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
	return nil
}

func Connect(ip string, mac string) error {
	_, err := nftQue.Exec(func() (bool, error) {
		nftMu.Lock()
		defer nftMu.Unlock()
		err := doConnect(ip, mac)
		return true, err
	})
	return err
}

func Disconnect(ip string, mac string) error {
	_, err := nftQue.Exec(func() (bool, error) {
		nftMu.Lock()
		defer nftMu.Unlock()
		err := doDisconnect(ip, mac)
		return true, err
	})
	return err
}

func IsConnected(mac string) bool {
	result, _ := nftQue.Exec(func() (bool, error) {
		nftMu.RLock()
		defer nftMu.RUnlock()
		return isConnected(mac), nil
	})
	return result
}

func AddInitCallback(cb func() error) {
	nftMu.Lock()
	defer nftMu.Unlock()
	initCallbacks = append(initCallbacks, cb)
}
