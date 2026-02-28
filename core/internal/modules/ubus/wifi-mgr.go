package ubus

import (
	"log"
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
		log.Printf("[WifiMgr] New listener registered (total listeners: %d)", len(self.listeners))
		retCh <- ch
	}()
	return <-retCh
}

// emit sends an event to all registered listeners (non-blocking)
func (self *WifiMgr) emit(event WifiEvent) {
	self.mu.Lock()
	defer self.mu.Unlock()

	if len(self.listeners) == 0 {
		log.Printf("[WifiMgr] WARNING: Event %s for MAC %s received but NO listeners registered!", event.Event, event.Mac)
		return
	}

	log.Printf("[WifiMgr] Emitting event %s for MAC %s to %d listeners", event.Event, event.Mac, len(self.listeners))

	activeListeners := []chan WifiEvent{}
	droppedCount := 0
	for i, ch := range self.listeners {
		select {
		case ch <- event:
			log.Printf("[WifiMgr] Event sent to listener %d/%d", i+1, len(self.listeners))
			activeListeners = append(activeListeners, ch)
		default:
			// Listener not consuming, close and remove
			log.Printf("[WifiMgr] WARNING: Listener %d/%d not consuming, dropping", i+1, len(self.listeners))
			close(ch)
			droppedCount++
		}
	}
	if droppedCount > 0 {
		log.Printf("[WifiMgr] Dropped %d non-consuming listeners, %d active remain", droppedCount, len(activeListeners))
	}
	self.listeners = activeListeners
}
