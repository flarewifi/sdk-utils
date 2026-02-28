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
}

func NewWifiMgr() *WifiMgr {
	return &WifiMgr{}
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
