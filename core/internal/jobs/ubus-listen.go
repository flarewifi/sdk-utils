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

// InitAndStartWifiMgr initializes the WiFi state tracker and starts the WiFi manager.
// This sets up shared state management for client connection tracking and starts
// both hostapd event listening and traffic-based fallback detection.
func InitAndStartWifiMgr(wifiMgr *ubus.WifiMgr) {
	// Initialize shared state tracker for connection state management
	wifiMgr.SetStateTracker(ubus.NewClientStateTracker())

	// Start WiFi event detection (hostapd + fallback)
	wifiMgr.Start()

	// StartWifiEventListener starts a goroutine that bridges WifiMgr events
	// to the legacy callback-based event handlers.
	// This should be called once during application boot.
	// Bridge WifiMgr events to legacy callbacks
	go func() {
		for evt := range wifiMgr.Listen() {
			api.EmitWifiEvent(evt.Event, evt.Mac)
		}
	}()
}
