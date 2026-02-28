//go:build dev

package ubus

import (
	"log"
)

// Start is a no-op in dev mode since hostapd is not available.
// This should be called once during application boot.
func (self *WifiMgr) Start() {
	log.Println("[WifiMgr] WiFi event listener skipped (dev mode - no hostapd available)")
}
