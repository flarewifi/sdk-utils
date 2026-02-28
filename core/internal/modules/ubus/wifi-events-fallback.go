//go:build !dev

package ubus

import (
	"log"
	"strings"
	"sync"
	"time"

	sdkapi "sdk/api"
)

const (
	// fallbackInactivityTimeout is the duration after which a client is considered disconnected
	// if no traffic is observed. This serves as a fallback when hostapd_cli misses events.
	fallbackInactivityTimeout = 2 * time.Minute

	// fallbackCheckInterval is how often we check for inactive clients
	fallbackCheckInterval = 30 * time.Second
)

// clientState tracks the state of a WiFi client for fallback detection
type clientState struct {
	lastActivity      time.Time
	disconnectEmitted bool // whether we've emitted a disconnect event for this inactivity period
}

// FallbackDetector detects WiFi client disconnect events based on traffic data.
// This runs in parallel with hostapd_cli to catch missed disconnect events.
// It only emits DISCONNECT events - connect events are handled by hostapd_cli.
type FallbackDetector struct {
	wifiMgr   *WifiMgr
	trafficCh <-chan sdkapi.TrafficData

	mu      sync.Mutex
	clients map[string]*clientState // MAC (uppercase) -> state

	stopCh chan struct{}
}

// NewFallbackDetector creates a new fallback detector
func NewFallbackDetector(wifiMgr *WifiMgr, trafficCh <-chan sdkapi.TrafficData) *FallbackDetector {
	return &FallbackDetector{
		wifiMgr:   wifiMgr,
		trafficCh: trafficCh,
		clients:   make(map[string]*clientState),
		stopCh:    make(chan struct{}),
	}
}

// Start begins the fallback detection loop
func (d *FallbackDetector) Start() {
	log.Println("[WifiMgr-Fallback] Starting traffic-based disconnect detection (parallel with hostapd_cli)")
	go d.run()
}

// Stop stops the fallback detector
func (d *FallbackDetector) Stop() {
	close(d.stopCh)
}

// run is the main loop that processes traffic data and checks for inactive clients
func (d *FallbackDetector) run() {
	checkTicker := time.NewTicker(fallbackCheckInterval)
	defer checkTicker.Stop()

	for {
		select {
		case <-d.stopCh:
			log.Println("[WifiMgr-Fallback] Stopping traffic-based disconnect detection")
			return

		case traffic, ok := <-d.trafficCh:
			if !ok {
				log.Println("[WifiMgr-Fallback] Traffic channel closed")
				return
			}
			d.processTraffic(traffic)

		case <-checkTicker.C:
			d.checkInactiveClients()
		}
	}
}

// processTraffic updates client activity based on traffic data
func (d *FallbackDetector) processTraffic(traffic sdkapi.TrafficData) {
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()

	// Process upload traffic (keyed by MAC address)
	for mac, stat := range traffic.Upload {
		if stat.Bytes > 0 || stat.Packets > 0 {
			mac = strings.ToUpper(mac)
			d.updateClientActivity(mac, now)
		}
	}
}

// updateClientActivity updates the activity time for a client
func (d *FallbackDetector) updateClientActivity(mac string, now time.Time) {
	state, exists := d.clients[mac]

	if !exists {
		// New client - just start tracking, don't emit connect
		// (hostapd_cli handles connect events)
		d.clients[mac] = &clientState{
			lastActivity:      now,
			disconnectEmitted: false,
		}
		return
	}

	// Existing client - update activity and reset disconnect flag
	state.lastActivity = now
	state.disconnectEmitted = false
}

// checkInactiveClients checks for clients that have been inactive and emits disconnect events
func (d *FallbackDetector) checkInactiveClients() {
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()

	for mac, state := range d.clients {
		if state.disconnectEmitted {
			// Already emitted disconnect for this inactivity period
			continue
		}

		inactiveDuration := now.Sub(state.lastActivity)
		if inactiveDuration >= fallbackInactivityTimeout {
			// Client has been inactive too long - emit disconnect
			state.disconnectEmitted = true
			log.Printf("[WifiMgr-Fallback] Client inactive for %v, emitting disconnect: %s",
				inactiveDuration.Round(time.Second), mac)
			d.emitDisconnect(mac)
		}
	}
}

// emitDisconnect emits a WiFi disconnect event through the WifiMgr
func (d *FallbackDetector) emitDisconnect(mac string) {
	d.wifiMgr.emit(WifiEvent{
		Interface: "fallback",
		Mac:       mac,
		Event:     sdkapi.WifiEventClientDisconnected,
	})
}
