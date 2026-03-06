//go:build !dev

package ubus

import (
	"log"
	"strings"
	"time"

	sdkapi "sdk/api"
)

const (
	// fallbackInactivityTimeout is the duration after which a client is considered disconnected
	// if no traffic is observed. This serves as a fallback when hostapd_cli misses events.
	fallbackInactivityTimeout = 15 * time.Minute

	// fallbackCheckInterval is how often we check for inactive clients
	fallbackCheckInterval = 30 * time.Second
)

// =============================================================================
// FALLBACK DETECTOR
// =============================================================================

// FallbackDetector detects WiFi client events based on traffic data.
// This runs in parallel with hostapd_cli to:
// 1. Catch missed disconnect events (when hostapd_cli fails)
// 2. Emit reconnect events when traffic resumes after fallback-detected disconnect
// It uses the shared ClientStateTracker in WifiMgr for state management.
type FallbackDetector struct {
	wifiMgr   *WifiMgr
	trafficCh <-chan sdkapi.TrafficData
	stopCh    chan struct{}
}

// NewFallbackDetector creates a new fallback detector
func NewFallbackDetector(wifiMgr *WifiMgr, trafficCh <-chan sdkapi.TrafficData) *FallbackDetector {
	return &FallbackDetector{
		wifiMgr:   wifiMgr,
		trafficCh: trafficCh,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the fallback detection loop
func (d *FallbackDetector) Start() {
	log.Println("[WifiMgr-Fallback] Starting traffic-based event detection (parallel with hostapd_cli)")
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
			log.Println("[WifiMgr-Fallback] Stopping traffic-based event detection")
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

// processTraffic updates client activity based on traffic data and emits reconnect events
func (d *FallbackDetector) processTraffic(traffic sdkapi.TrafficData) {
	stateTracker := d.wifiMgr.stateTracker
	if stateTracker == nil {
		return // State tracker not initialized yet
	}

	// Process upload traffic (keyed by MAC address)
	for mac, stat := range traffic.Upload {
		if stat.Bytes > 0 || stat.Packets > 0 {
			mac = strings.ToUpper(mac)

			// Check if this traffic indicates a reconnection
			shouldEmitConnect := stateTracker.OnTrafficDetected(mac)

			if shouldEmitConnect {
				// Client was disconnected, now has traffic - emit connect event
				log.Printf("[WifiMgr-Fallback] Client resumed activity after disconnect, emitting connect: %s", mac)
				d.emitConnect(mac)
			}
		}
	}
}

// checkInactiveClients checks for clients that have been inactive and emits disconnect events
func (d *FallbackDetector) checkInactiveClients() {
	stateTracker := d.wifiMgr.stateTracker
	if stateTracker == nil {
		return // State tracker not initialized yet
	}

	// Get list of MACs that should be marked inactive
	inactiveMACs := stateTracker.CheckInactivity(fallbackInactivityTimeout)

	for _, mac := range inactiveMACs {
		// Attempt to mark as inactive (returns true if state changed)
		if stateTracker.MarkInactive(mac) {
			log.Printf("[WifiMgr-Fallback] Client inactive for %v, emitting disconnect: %s",
				fallbackInactivityTimeout, mac)
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

// emitConnect emits a WiFi connect event through the WifiMgr
func (d *FallbackDetector) emitConnect(mac string) {
	d.wifiMgr.emit(WifiEvent{
		Interface: "fallback",
		Mac:       mac,
		Event:     sdkapi.WifiEventClientConnected,
	})
}
