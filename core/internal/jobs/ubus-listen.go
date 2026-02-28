package jobs

import (
	"log"

	"core/internal/api"
	"core/internal/modules/ubus"
)

// StartUbusListener starts the ubus event listener for network interface events.
// This should be called once during application boot.
func StartUbusListener() {
	log.Println("[Jobs] Starting ubus listener for network interface events...")
	ubus.Listen()
	log.Println("[Jobs] Ubus listener started")
}

// StartWifiEventListener starts a goroutine that bridges WifiMgr events
// to the legacy callback-based event handlers.
// This should be called once during application boot.
func StartWifiEventListener(wifiMgr *ubus.WifiMgr) {
	log.Println("[Jobs] Starting WiFi event listener bridge...")
	// Bridge WifiMgr events to legacy callbacks
	go func() {
		log.Println("[Jobs] WiFi event bridge goroutine started, waiting for events...")
		eventCount := 0
		for evt := range wifiMgr.Listen() {
			eventCount++
			log.Printf("[Jobs] WiFi event #%d received: %s for MAC %s, forwarding to handlers...", eventCount, evt.Event, evt.Mac)
			api.EmitWifiEvent(evt.Event, evt.Mac)
			log.Printf("[Jobs] WiFi event #%d forwarded to handlers", eventCount)
		}
		log.Println("[Jobs] WiFi event bridge channel closed")
	}()
	log.Println("[Jobs] WiFi event listener bridge initialized")
}
