package ubus

import (
	"sync"

	sdkapi "sdk/api"
)

// WifiEvent represents a WiFi client event
type WifiEvent struct {
	Interface string
	Mac       string
	Event     sdkapi.WifiClientEvent
}

// WifiMgr manages WiFi event listeners following the TrafficMgr pattern
type WifiMgr struct {
	mu        sync.RWMutex
	listeners []chan WifiEvent

	// trafficCh is used for fallback detection when hostapd_cli is unavailable
	trafficCh <-chan sdkapi.TrafficData

	// stateTracker provides shared state management across hostapd and fallback detection.
	// This ensures consistent connection state and prevents duplicate events.
	stateTracker *ClientStateTracker
}

func NewWifiMgr() *WifiMgr {
	return &WifiMgr{}
}

// SetStateTracker sets the shared client state tracker.
// This must be called before Start() to enable state tracking.
func (self *WifiMgr) SetStateTracker(tracker *ClientStateTracker) {
	self.stateTracker = tracker
}

// StateTracker returns the shared client state tracker
func (self *WifiMgr) StateTracker() *ClientStateTracker {
	return self.stateTracker
}

// SetTrafficChannel sets the traffic data channel for fallback detection.
// This must be called before Start() to enable fallback detection.
func (self *WifiMgr) SetTrafficChannel(ch <-chan sdkapi.TrafficData) {
	self.trafficCh = ch
}

// Start begins listening for WiFi events.
// Implementation is in wifi-events_prod.go (production) or wifi-events_dev.go (development).

// Listen registers a new listener and returns a channel for receiving events
func (self *WifiMgr) Listen() <-chan WifiEvent {
	retCh := make(chan chan WifiEvent)
	go func() {
		self.mu.Lock()
		defer self.mu.Unlock()
		ch := make(chan WifiEvent)
		self.listeners = append(self.listeners, ch)
		retCh <- ch
	}()
	return <-retCh
}

// emit sends an event to all registered listeners (non-blocking)
func (self *WifiMgr) emit(event WifiEvent) {
	self.mu.Lock()
	defer self.mu.Unlock()

	if len(self.listeners) == 0 {
		return
	}

	activeListeners := []chan WifiEvent{}
	for _, ch := range self.listeners {
		select {
		case ch <- event:
			activeListeners = append(activeListeners, ch)
		default:
			// Listener not consuming, close and remove
			close(ch)
		}
	}
	self.listeners = activeListeners
}
