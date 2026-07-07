package jobs

import (
	"context"

	"core/internal/api"
	"core/internal/modules/ubus"

	sdkapi "sdk/api"
)

// StartUbusListener starts the ubus event listener for network interface events.
// This should be called once during application boot.
func StartUbusListener() {
	ubus.Listen()
}

// InitAndStartWifiMgr initializes the WiFi state tracker and starts the WiFi manager.
// This sets up shared state management for client connection tracking and starts
// both hostapd event listening and traffic-based fallback detection.
func InitAndStartWifiMgr(scheduler sdkapi.ISchedulerApi, wifiMgr *ubus.WifiMgr) error {
	// Initialize shared state tracker for connection state management
	wifiMgr.SetStateTracker(ubus.NewClientStateTracker())

	// Start WiFi event detection (hostapd + fallback)
	wifiMgr.Start()

	// Bridge WifiMgr events to the legacy callback-based event handlers, until
	// shutdown or the events channel closes.
	return scheduler.Go("wifi-event-bridge", func(ctx context.Context) {
		events := wifiMgr.Listen()
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-events:
				if !ok {
					return
				}
				api.EmitWifiEvent(evt.Event, evt.Mac)
			}
		}
	})
}
