//go:build dev

package ubus

// Start is a no-op in dev mode since hostapd is not available.
// This should be called once during application boot.
func (self *WifiMgr) Start() {
}
