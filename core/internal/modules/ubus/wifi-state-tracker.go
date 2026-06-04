package ubus

import (
	"sync"
	"time"
)

const (
	// staleClientCleanupInterval is how often we check for stale clients to remove
	staleClientCleanupInterval = 1 * time.Hour

	// staleClientTimeout is how long a disconnected client stays in memory before removal
	staleClientTimeout = 24 * time.Hour
)

// =============================================================================
// CLIENT STATE TRACKER
// =============================================================================

// ClientConnectionState represents the known connection state of a WiFi client
type ClientConnectionState string

const (
	StateConnected    ClientConnectionState = "connected"    // Client is connected
	StateDisconnected ClientConnectionState = "disconnected" // Client is disconnected
	StateUnknown      ClientConnectionState = "unknown"      // Initial state
)

// ClientState tracks comprehensive state for a single client
type ClientState struct {
	mac              string
	state            ClientConnectionState
	lastActivity     time.Time
	lastStateChange  time.Time
	disconnectSource string // "hostapd", "fallback", or ""
}

// ClientStateTracker tracks WiFi client states across both hostapd and traffic detection.
// This provides a single source of truth for connection state, preventing duplicate events
// and enabling reconnect detection via traffic monitoring.
type ClientStateTracker struct {
	mu      sync.RWMutex
	clients map[string]*ClientState // MAC (uppercase) -> state
}

// NewClientStateTracker creates a new client state tracker
func NewClientStateTracker() *ClientStateTracker {
	tracker := &ClientStateTracker{
		clients: make(map[string]*ClientState),
	}

	// Start cleanup goroutine
	go tracker.cleanupStaleClients()

	return tracker
}

// OnHostapdConnect is called when hostapd emits a connect event.
// Returns true if state changed (should emit event).
func (t *ClientStateTracker) OnHostapdConnect(mac string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	state, exists := t.clients[mac]

	if !exists {
		// New client
		t.clients[mac] = &ClientState{
			mac:             mac,
			state:           StateConnected,
			lastActivity:    now,
			lastStateChange: now,
		}
		return true
	}

	if state.state == StateConnected {
		// Already connected - duplicate event
		state.lastActivity = now
		return false
	}

	// Reconnecting - state change
	state.state = StateConnected
	state.lastActivity = now
	state.lastStateChange = now
	state.disconnectSource = ""
	return true
}

// OnHostapdDisconnect is called when hostapd emits a disconnect event.
// Returns true if state changed (should emit event).
func (t *ClientStateTracker) OnHostapdDisconnect(mac string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	state, exists := t.clients[mac]

	if !exists {
		// Unknown client disconnecting - create entry
		t.clients[mac] = &ClientState{
			mac:              mac,
			state:            StateDisconnected,
			lastActivity:     now,
			lastStateChange:  now,
			disconnectSource: "hostapd",
		}
		return true
	}

	if state.state == StateDisconnected {
		// Already disconnected - duplicate event
		return false
	}

	// State change to disconnected
	state.state = StateDisconnected
	state.lastStateChange = now
	state.disconnectSource = "hostapd"
	return true
}

// OnTrafficDetected is called when traffic is observed for a MAC.
// Returns true if this indicates a reconnection (should emit connect event).
func (t *ClientStateTracker) OnTrafficDetected(mac string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	state, exists := t.clients[mac]

	if !exists {
		// Unknown client with traffic - assume hostapd missed connect event
		// Create as Connected and return true to emit event
		t.clients[mac] = &ClientState{
			mac:             mac,
			state:           StateConnected,
			lastActivity:    now,
			lastStateChange: now,
		}
		return true
	}

	// Update activity time
	state.lastActivity = now

	if state.state == StateDisconnected {
		// Client was disconnected, now has traffic - reconnection!
		state.state = StateConnected
		state.lastStateChange = now
		state.disconnectSource = ""
		return true
	}

	// Already connected - just activity update
	return false
}

// CheckInactivity returns list of MACs that should be marked as inactive.
// Only returns MACs currently in Connected state with lastActivity exceeding timeout.
func (t *ClientStateTracker) CheckInactivity(timeout time.Duration) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	var inactiveMACs []string

	for mac, state := range t.clients {
		if state.state == StateConnected {
			inactiveDuration := now.Sub(state.lastActivity)
			if inactiveDuration >= timeout {
				inactiveMACs = append(inactiveMACs, mac)
			}
		}
	}

	return inactiveMACs
}

// MarkInactive marks a client as disconnected due to inactivity.
// Returns true if state changed (should emit event).
func (t *ClientStateTracker) MarkInactive(mac string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	state, exists := t.clients[mac]
	if !exists {
		// Should not happen, but handle gracefully
		return false
	}

	if state.state == StateDisconnected {
		// Already disconnected
		return false
	}

	// Mark as disconnected
	now := time.Now()
	state.state = StateDisconnected
	state.lastStateChange = now
	state.disconnectSource = "fallback"
	return true
}

// cleanupStaleClients runs periodically to remove old disconnected clients from memory
func (t *ClientStateTracker) cleanupStaleClients() {
	ticker := time.NewTicker(staleClientCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		t.mu.Lock()
		now := time.Now()

		for mac, state := range t.clients {
			if state.state == StateDisconnected {
				staleDuration := now.Sub(state.lastStateChange)
				if staleDuration >= staleClientTimeout {
					delete(t.clients, mac)
				}
			}
		}

		t.mu.Unlock()
	}
}

// GetClientCount returns the current number of tracked clients (for debugging)
func (t *ClientStateTracker) GetClientCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.clients)
}

// GetConnectedCount returns the number of currently connected clients (for debugging)
func (t *ClientStateTracker) GetConnectedCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	count := 0
	for _, state := range t.clients {
		if state.state == StateConnected {
			count++
		}
	}
	return count
}
