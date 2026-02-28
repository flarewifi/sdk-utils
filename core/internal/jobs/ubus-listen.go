package jobs

import (
	"core/internal/api"
	"core/internal/modules/ubus"
)

// StartUbusListener starts the ubus event listener for network interface events.
// This should be called once during application boot.
func StartUbusListener() {
	ubus.Listen()
}

// StartWifiEventListener starts a goroutine that bridges WifiMgr events
// to the legacy callback-based event handlers.
// This should be called once during application boot.
func StartWifiEventListener(wifiMgr *ubus.WifiMgr) {
	// Bridge WifiMgr events to legacy callbacks
	go func() {
		for evt := range wifiMgr.Listen() {
			api.EmitWifiEvent(evt.Event, evt.Mac)
		}
	}()
}
