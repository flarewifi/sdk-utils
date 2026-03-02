package sessmgr

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"core/internal/modules/tc"
	"core/internal/network"
	sdkapi "sdk/api"
)

const (
	// bytesPerMiB is the number of bytes in a mebibyte (1024^2)
	bytesPerMiB = 1024 * 1024
)

var (
	// tcNftMu serializes all TC and NFT commands globally.
	// TC/NFT subsystem can only handle one command at a time.
	tcNftMu           sync.Mutex
	ErrSessionExpired = errors.New("session expired")
	ErrSessionStopped = errors.New("session already stopped")
)

// logSlowOperation logs a warning if an operation exceeds the threshold duration.
func logSlowOperation(operation string, start time.Time, threshold time.Duration, context string) {
	elapsed := time.Since(start)
	if elapsed > threshold {
		log.Printf("[SLOW] %s took %v (threshold: %v) - %s", operation, elapsed, threshold, context)
	}
}

// withTcNftLock executes a function while holding the global TC/NFT lock.
// Logs debug info about lock acquisition and operation duration.
func withTcNftLock(operation string, context string, fn func() error) error {
	lockStart := time.Now()
	tcNftMu.Lock()
	lockWait := time.Since(lockStart)
	if lockWait > 500*time.Millisecond {
		log.Printf("[SLOW] Waiting for tcNftMu took %v - %s - %s", lockWait, operation, context)
	}

	opStart := time.Now()
	err := fn()
	tcNftMu.Unlock()

	logSlowOperation(operation, opStart, 1*time.Second, context)
	return err
}

// SessionEventEmitter interface for emitting session events
type SessionEventEmitter interface {
	emitSessionEvent(event sdkapi.SessionEvent, session sdkapi.IClientSession, device sdkapi.IClientDevice) error
}

// networkState holds network-related fields that rarely change.
// Uses atomic pointer for lock-free reads.
type networkState struct {
	ip  string
	mac string
	lan *network.NetworkLan
}

func NewRunningSession(clnt sdkapi.IClientDevice, s sdkapi.IClientSession, emitter SessionEventEmitter) (*RunningSession, error) {
	log.Printf("[Running Session] Creating new running session - DeviceID=%d, MAC=%s, IP=%s",
		clnt.ID(), clnt.MacAddr(), clnt.IpAddr())

	lan, err := network.FindByIp(clnt.IpAddr())
	if err != nil {
		log.Printf("[Running Session] ERROR - Failed to find LAN for IP %s: %v", clnt.IpAddr(), err)
		return nil, err
	}

	rs := &RunningSession{
		// Immutable fields - set once, never change
		clnt:    clnt,
		clntId:  clnt.ID(),
		emitter: emitter,

		// Mutable state initialized
		session:   s,
		callbacks: []chan error{},
	}

	// Initialize network state atomically
	rs.network.Store(&networkState{
		ip:  clnt.IpAddr(),
		mac: clnt.MacAddr(),
		lan: lan,
	})

	log.Printf("[Running Session] Running session created successfully - DeviceID=%d, MAC=%s, IP=%s, LAN=%s",
		rs.clntId, clnt.MacAddr(), clnt.IpAddr(), lan.Name())

	return rs, nil
}

// RunningSession manages an active client session with bandwidth control.
//
// Field categories:
//   - IMMUTABLE: clnt, clntId, emitter - set at creation, never change, no lock needed
//   - ATOMIC: network - rarely changes, uses atomic.Pointer for lock-free reads
//   - MUTABLE: everything else - protected by mu
type RunningSession struct {
	// === IMMUTABLE after creation (no lock needed) ===
	clnt    sdkapi.IClientDevice
	clntId  int64
	emitter SessionEventEmitter

	// === RARELY CHANGES (atomic pointer for lock-free reads) ===
	network atomic.Pointer[networkState]

	// === MUTABLE STATE (protected by mu) ===
	mu            sync.Mutex // Changed from RWMutex - simpler, less error-prone
	tcClassId     *tc.TcClassId
	timeTimer     *time.Timer
	timerCancel   context.CancelFunc
	timerCtx      context.Context
	timerGen      uint64 // Generation counter to prevent stale timer goroutines from acting
	session       sdkapi.IClientSession
	diffMb        float64
	callbacks     []chan error
	stopped       bool // Prevents duplicate Stop() operations; cleared only by Start()
	saveFailCount int  // Consecutive periodic save failures; reset on success
}

// ============================================================================
// LOCK-FREE ACCESSORS - Immutable fields, no synchronization needed
// ============================================================================

// ClientId returns the device ID (immutable, no lock needed).
func (self *RunningSession) ClientId() int64 {
	return self.clntId
}

// ============================================================================
// ATOMIC ACCESSORS - Network state uses atomic pointer
// ============================================================================

// IpAddr returns the current IP address (atomic read, no lock needed).
func (self *RunningSession) IpAddr() string {
	return self.network.Load().ip
}

// MacAddr returns the current MAC address (atomic read, no lock needed).
func (self *RunningSession) MacAddr() string {
	return self.network.Load().mac
}

// Lan returns the current LAN interface (atomic read, no lock needed).
func (self *RunningSession) Lan() *network.NetworkLan {
	return self.network.Load().lan
}

// ============================================================================
// MUTEX-PROTECTED ACCESSORS - Mutable state
// ============================================================================

// GetSession returns the current session (requires lock).
func (self *RunningSession) GetSession() sdkapi.IClientSession {
	self.mu.Lock()
	defer self.mu.Unlock()
	return self.session
}

// Done returns a channel that receives when the session ends.
// Safe to call multiple times - each caller gets their own channel.
func (self *RunningSession) Done() <-chan error {
	self.mu.Lock()
	defer self.mu.Unlock()

	// Use buffered channel to prevent StopWithReason() from blocking
	ch := make(chan error, 1)

	// If already stopped, return immediately with appropriate error
	if self.stopped {
		ch <- ErrSessionStopped
		return ch
	}

	self.callbacks = append(self.callbacks, ch)
	return ch
}

// DiffMb returns the unsaved data consumption difference.
func (self *RunningSession) DiffMb() (mb float64) {
	self.mu.Lock()
	defer self.mu.Unlock()
	return self.diffMb
}

// IsStopped returns whether the session has been stopped.
func (self *RunningSession) IsStopped() bool {
	self.mu.Lock()
	defer self.mu.Unlock()
	return self.stopped
}

// Reset prepares the RunningSession for reuse with a new session.
// Preserves TC class/filter to avoid teardown/recreation overhead.
// Must be called after StopWithReason() and before Start() with new session.
//
// NOTE: stopped flag remains true until Start() successfully loads a new session.
// This ensures that if Stop() is called after Reset() but before Start() (e.g., when
// no more sessions are available), it's a no-op since the session was already stopped.
func (self *RunningSession) Reset() {
	self.mu.Lock()
	defer self.mu.Unlock()

	// Note: stopped remains true - only Start() clears it when new session loads
	self.callbacks = []chan error{}
	self.diffMb = 0
	self.saveFailCount = 0
	// Note: session is intentionally preserved until Start() sets a new one
	// Note: tcClassId is intentionally preserved for reuse
	// Note: network state is intentionally preserved
}

// ============================================================================
// NETWORK UPDATE - Atomic swap with TC operations
// ============================================================================

// UpdateNetworkDetails updates the MAC and IP address when device network details change.
func (self *RunningSession) UpdateNetworkDetails(ctx context.Context, newMac, newIP string) error {
	contextInfo := fmt.Sprintf("DeviceID=%d, NewMAC=%s, NewIP=%s",
		self.clntId, newMac, newIP)

	log.Printf("[Running Session] UpdateNetworkDetails - %s", contextInfo)

	// Quick check outside lock - if nothing changed, skip entirely.
	// This is an optimization; the authoritative check happens inside the lock.
	quickNet := self.network.Load()
	if quickNet.ip == newIP && quickNet.mac == newMac {
		log.Printf("[Running Session] No network changes detected for device %d", self.clntId)
		return nil
	}

	// Determine new LAN outside lock (network.FindByIp is read-only and safe)
	var newLan *network.NetworkLan
	if quickNet.ip != newIP {
		log.Printf("[Running Session] IP changed, checking if LAN changed...")
		var err error
		newLan, err = network.FindByIp(newIP)
		if err != nil {
			log.Printf("[Running Session] ERROR - Failed to find LAN for new IP %s: %v", newIP, err)
			return err
		}
	}

	// All TC operations and network state updates happen inside tcNftMu+mu.
	// Re-read network state inside the lock to handle concurrent UpdateNetworkDetails calls.
	// If another call already updated the state, we detect it and adjust accordingly.
	return withTcNftLock("TC Network Update", contextInfo, func() error {
		self.mu.Lock()
		defer self.mu.Unlock()

		// Re-read current network state under lock (authoritative check)
		currentNet := self.network.Load()

		// Check if update is still needed (another concurrent call may have already done it)
		if currentNet.ip == newIP && currentNet.mac == newMac {
			log.Printf("[Running Session] Network already updated by concurrent call for device %d, skipping", self.clntId)
			return nil
		}

		// If IP hasn't changed (only MAC), just update network state
		if currentNet.ip == newIP {
			self.network.Store(&networkState{
				ip:  newIP,
				mac: newMac,
				lan: currentNet.lan,
			})
			return nil
		}

		// IP changed - need to update TC rules
		// Use newLan computed above (or fall back to current LAN if IP didn't change relative to quickNet)
		if newLan == nil {
			newLan = currentNet.lan
		}

		if newLan.Name() != currentNet.lan.Name() {
			// LAN changed - need to recreate TC rules on new interface
			log.Printf("[Running Session] LAN changed from %s to %s, recreating TC rules...",
				currentNet.lan.Name(), newLan.Name())

			// Clean up old TC rules on current (old) LAN
			if self.tcClassId != nil {
				classid := self.tcClassId.Uint()
				log.Printf("[Running Session] Removing old TC filter for IP %s", currentNet.ip)
				if err := currentNet.lan.DelFilter(currentNet.ip, classid); err != nil {
					log.Printf("[Running Session] WARNING - Failed to delete old filter: %v", err)
				}

				log.Printf("[Running Session] Removing old TC class %d", classid)
				if err := currentNet.lan.DelClass(classid); err != nil {
					log.Printf("[Running Session] WARNING - Failed to delete old class: %v", err)
				}
				self.tcClassId = nil
			}

			// Update network state BEFORE initTc so it reads the new LAN
			self.network.Store(&networkState{
				ip:  newIP,
				mac: newMac,
				lan: newLan,
			})

			// Recreate TC rules on new interface
			log.Printf("[Running Session] Creating new TC rules on interface %s", newLan.Name())
			if err := self.initTc(); err != nil {
				log.Printf("[Running Session] ERROR - Failed to create TC rules: %v", err)
				return err
			}
			log.Printf("[Running Session] TC rules recreated successfully")
		} else {
			// Same LAN, just update the filter
			log.Printf("[Running Session] Same LAN, updating TC filter from IP %s to %s", currentNet.ip, newIP)

			if self.tcClassId != nil {
				classid := self.tcClassId.Uint()

				// Remove old filter using current (authoritative) IP
				if err := currentNet.lan.DelFilter(currentNet.ip, classid); err != nil {
					log.Printf("[Running Session] WARNING - Failed to delete old filter: %v", err)
				}

				// Create new filter with new IP
				if err := newLan.CreateFilter(newIP, classid); err != nil {
					log.Printf("[Running Session] ERROR - Failed to create new filter: %v", err)
					return err
				}
				log.Printf("[Running Session] TC filter updated successfully")
			}

			// Update network state AFTER TC filter is ready
			self.network.Store(&networkState{
				ip:  newIP,
				mac: newMac,
				lan: newLan,
			})
		}

		return nil
	})
}

// ============================================================================
// SESSION LIFECYCLE
// ============================================================================

func (self *RunningSession) Start(ctx context.Context, s sdkapi.IClientSession) error {
	net := self.network.Load()
	contextInfo := fmt.Sprintf("DeviceID=%d, SessionID=%d, MAC=%s, IP=%s",
		self.clntId, s.ID(), net.mac, net.ip)

	log.Printf("[Running Session] Start - %s", contextInfo)

	// 1. DB reload - no lock needed (session has its own synchronization)
	dbStart := time.Now()
	if err := s.Reload(ctx); err != nil {
		return fmt.Errorf("failed to reload session: %w", err)
	}
	logSlowOperation("DB Reload", dbStart, 2*time.Second, contextInfo)

	// 2. Update session state atomically
	// Note: We allow Start() even if stopped=true (after Reset()) to enable session chaining
	self.mu.Lock()
	self.stopped = false // Clear stopped flag for new session
	self.session = s

	// Set first start time if this is the first time session is starting
	// and set resumed time to track current running period
	timeNow := time.Now().UTC()
	updateData := sdkapi.SessionUpdateData{}
	hasUpdates := false

	if s.StartedAt() == nil {
		updateData.StartedAt = &timeNow
		hasUpdates = true
	}

	if s.ResumedAt() == nil {
		updateData.ResumedAt = &timeNow
		hasUpdates = true
	}

	// Apply timestamp updates in batch if needed
	if hasUpdates {
		s.SetData(updateData)
	}
	self.mu.Unlock()

	// 3. Save to DB - no lock needed
	dbStart = time.Now()
	if err := s.Save(ctx); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	logSlowOperation("DB Save", dbStart, 2*time.Second, contextInfo)

	// 4. TC operations - check and init/update inside single lock acquisition
	err := withTcNftLock("TC Init/Update", contextInfo, func() error {
		self.mu.Lock()
		defer self.mu.Unlock()

		if self.tcClassId == nil {
			return self.initTc()
		}
		return self.updateTc()
	})
	if err != nil {
		return fmt.Errorf("failed to setup TC: %w", err)
	}

	// 5. Start timer if not already running
	self.mu.Lock()
	if self.timeTimer == nil {
		self.initTimeTimer(s)
		log.Println("Session timer has started...")
	}
	self.mu.Unlock()

	log.Printf("[Running Session] Start completed - %s", contextInfo)
	return nil
}

func (self *RunningSession) Stop(ctx context.Context) error {
	return self.StopWithReason(ctx, false)
}

func (self *RunningSession) StopWithReason(ctx context.Context, isConsumed bool) error {
	// First, mark as stopped and collect state atomically
	self.mu.Lock()
	if self.stopped {
		self.mu.Unlock()
		log.Printf("[Running Session] StopWithReason - already stopped, DeviceID=%d", self.clntId)
		return nil
	}
	self.stopped = true

	session := self.session
	callbacks := self.callbacks
	self.callbacks = nil // Clear callbacks

	// Clean up timer while holding lock
	self.cleanUpTimer()
	self.mu.Unlock()

	// Build context info for logging (after releasing lock)
	var sessionID int64
	if session != nil {
		sessionID = session.ID()
	}
	contextInfo := fmt.Sprintf("DeviceID=%d, SessionID=%d, Expired=%v",
		self.clntId, sessionID, isConsumed)

	log.Printf("[Running Session] StopWithReason - %s", contextInfo)

	// Calculate and record elapsed time
	elapsed := self.snapshotTimeConsumption(session, true)
	if elapsed > 0 && session != nil {
		log.Printf("Recording elapsed time: %d seconds (total consumption: %d)\n",
			elapsed, session.TimeConsumption())
	}

	// DB save - no lock needed (session has its own synchronization)
	dbStart := time.Now()
	saveErr := self.persistSession(ctx, session)
	logSlowOperation("DB Save (Stop)", dbStart, 2*time.Second, contextInfo)

	// Emit events (emitter is immutable, no lock needed)
	// Only emit events if we have a valid session - session can be nil if Stop() is called
	// before Start() completes or if the session was never properly initialized
	if self.emitter != nil && self.clnt != nil && session != nil {
		if isConsumed {
			// Session was consumed (time/data exhausted) - emit both consumed and disconnected
			self.emitter.emitSessionEvent(sdkapi.EventSessionConsumed, session, self.clnt)
		}
		// Always emit disconnected event when session stops
		self.emitter.emitSessionEvent(sdkapi.EventSessionDisconnected, session, self.clnt)
	}

	// Determine the error to return to callbacks
	var callbackErr error
	if isConsumed {
		callbackErr = ErrSessionExpired
		// Log save error if it occurred during consumed stop
		// (data loss is already mitigated by periodic saves - at most 1 minute lost)
		if saveErr != nil {
			log.Printf("[Running Session] WARNING - Save failed during consumed stop, consumption data may be lost: %v", saveErr)
		}
	} else {
		callbackErr = saveErr
	}

	// Send to callbacks - no lock needed (we own the slice)
	for _, cb := range callbacks {
		select {
		case cb <- callbackErr:
		default:
			// Channel full or closed, skip
		}
	}
	log.Println("Done running callbacks.")

	log.Printf("[Running Session] StopWithReason completed - %s", contextInfo)
	return callbackErr
}

func (self *RunningSession) CleanupTc() error {
	return withTcNftLock("CleanupTc", fmt.Sprintf("DeviceID=%d", self.clntId), func() error {
		self.mu.Lock()
		defer self.mu.Unlock()

		// Read network state inside lock to ensure consistency with UpdateNetworkDetails
		net := self.network.Load()

		if self.tcClassId != nil {
			log.Printf("Clean up TC... DeviceID=%d, IP=%s", self.clntId, net.ip)
			classid := *self.tcClassId

			if err := net.lan.DelFilter(net.ip, classid.Uint()); err != nil {
				return err
			}

			if err := net.lan.DelClass(classid.Uint()); err != nil {
				return err
			}

			self.tcClassId = nil
		}

		log.Println("Done cleaning TC.")
		return nil
	})
}

// ============================================================================
// DATA CONSUMPTION
// ============================================================================

func (self *RunningSession) UpdateDataConsumption(stats *sdkapi.TrafficData) {
	// Check if stopped first and grab session reference atomically
	self.mu.Lock()
	if self.stopped {
		self.mu.Unlock()
		return
	}
	session := self.session
	if session == nil {
		self.mu.Unlock()
		return
	}
	self.mu.Unlock()

	// Get network state atomically (no lock needed)
	net := self.network.Load()

	// Look up traffic stats
	download, dlOK := stats.Download[net.ip]
	upload, upOK := stats.Upload[strings.ToUpper(net.mac)]

	if !dlOK || !upOK {
		return
	}

	dataconMb := float64(download.Bytes+upload.Bytes) / bytesPerMiB
	log.Println("CONSUMPTION MiB: ", dataconMb)

	// Update session data consumption (session has its own synchronization)
	session.IncDataCons(dataconMb)

	// Update diffMb and check if we should trigger stop.
	// Check stopped again under lock to prevent spawning multiple StopWithReason goroutines.
	self.mu.Lock()
	if self.stopped {
		self.mu.Unlock()
		return
	}
	self.diffMb += dataconMb
	shouldStop := session.IsConsumed()
	self.mu.Unlock()

	if shouldStop {
		log.Println("Session data is consumed!!!")
		// StopWithReason has its own stopped guard, so concurrent calls are safe (only first wins)
		go self.StopWithReason(context.Background(), true)
	}
}

// ============================================================================
// INTERNAL HELPERS - Must be called with appropriate locks held
// ============================================================================

// initTimeTimer initializes the session timer. Must be called with mu held.
func (self *RunningSession) initTimeTimer(s sdkapi.IClientSession) {
	// Check if session is already consumed or expired
	if s.IsConsumed() {
		log.Println("Session already consumed or expired, stopping immediately")
		// Use goroutine to avoid calling StopWithReason while mu is held.
		// StopWithReason has its own stopped guard so concurrent calls are safe.
		go self.StopWithReason(context.Background(), true)
		return
	}

	// Calculate remaining time and start timer
	remainingSecs := s.RemainingTime()
	self.startTimer(remainingSecs)
}

// startTimer creates and starts a new timer with the specified duration.
// Must be called with mu held.
// Assumes any existing timer has been cleaned up.
func (self *RunningSession) startTimer(remainingSecs int) {
	ctx, cancel := context.WithCancel(context.Background())
	self.timerCtx = ctx
	self.timerCancel = cancel

	// Increment generation so any stale timer goroutines become no-ops
	self.timerGen++
	gen := self.timerGen

	duration := time.Duration(remainingSecs) * time.Second
	timer := time.NewTimer(duration)
	self.timeTimer = timer

	log.Printf("[Running Session] Timer started for %d seconds, DeviceID=%d, gen=%d", remainingSecs, self.clntId, gen)

	go self.timerLoop(ctx, timer, gen)
}

// timerLoop handles timer expiration and periodic saves.
// Runs in a separate goroutine. The gen parameter ensures stale goroutines
// (from a previous timer that was cancelled and replaced) become no-ops.
func (self *RunningSession) timerLoop(ctx context.Context, timer *time.Timer, gen uint64) {
	saveTicker := time.NewTicker(1 * time.Minute)
	defer saveTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Session timer cancelled")
			return

		case <-timer.C:
			// Verify this timer goroutine is still current before acting
			self.mu.Lock()
			if self.timerGen != gen || self.stopped {
				self.mu.Unlock()
				log.Printf("[Running Session] Stale timer expired (gen=%d, current=%d), ignoring", gen, self.timerGen)
				return
			}
			self.mu.Unlock()

			log.Println("Session timer expired - time consumed!")
			self.StopWithReason(context.Background(), true)
			return

		case <-saveTicker.C:
			if !self.handlePeriodicSave(gen) {
				return
			}
		}
	}
}

// handlePeriodicSave persists time consumption and emits update event.
// The gen parameter is the timer generation that spawned this periodic save.
// Returns true to continue the timer loop, false to exit.
func (self *RunningSession) handlePeriodicSave(gen uint64) bool {
	self.mu.Lock()
	if self.stopped || self.timerGen != gen {
		self.mu.Unlock()
		return false
	}
	currentSession := self.session
	// Reset diffMb now - any new consumption after this point will be tracked in the next period.
	// This prevents data loss from consumption arriving between save and reset.
	self.diffMb = 0
	self.mu.Unlock()

	if currentSession == nil {
		return true
	}

	// Persist time consumption to protect against crashes
	// This ensures at most 1 minute of time tracking is lost on crash
	// instead of all time since session start
	elapsed := self.snapshotTimeConsumption(currentSession, false)
	log.Printf("Periodic save: persisting %d seconds consumed, %d remaining\n",
		currentSession.ConsumedTimeSecs(), currentSession.RemainingTime())

	contextInfo := fmt.Sprintf("DeviceID=%d, SessionID=%d, Elapsed=%ds",
		self.clntId, currentSession.ID(), elapsed)

	dbStart := time.Now()
	saveErr := self.persistSession(context.Background(), currentSession)
	logSlowOperation("Periodic Save", dbStart, 2*time.Second, contextInfo)

	if saveErr != nil {
		// Tolerate transient DB errors - only stop after consecutive failures.
		// This prevents a single slow/failed DB write from permanently disconnecting the device.
		// At most 1 minute of consumption data is at risk per failed save (periodic save interval).
		self.mu.Lock()
		self.saveFailCount++
		failCount := self.saveFailCount
		self.mu.Unlock()

		const maxConsecutiveFailures = 3
		if failCount >= maxConsecutiveFailures {
			log.Printf("[ERROR] Periodic save failed %d consecutive times: %v - STOPPING SESSION", failCount, saveErr)
			go self.Stop(context.Background())
			return false
		}
		log.Printf("[WARN] Periodic save failed (%d/%d): %v - will retry next period", failCount, maxConsecutiveFailures, saveErr)
		return true
	}

	// Save succeeded - reset failure counter
	self.mu.Lock()
	self.saveFailCount = 0
	self.mu.Unlock()

	// Emit session:updated event - re-read session under lock to avoid stale reference
	self.mu.Lock()
	freshSession := self.session
	stale := self.stopped || self.timerGen != gen
	self.mu.Unlock()

	if stale {
		return false
	}

	if self.emitter != nil && self.clnt != nil && freshSession != nil {
		self.emitter.emitSessionEvent(sdkapi.EventSessionUpdated, freshSession, self.clnt)
	}

	// Check if session is now consumed or expired (e.g., expiration date passed)
	if freshSession != nil && freshSession.IsConsumed() {
		log.Println("Session consumed or expired during periodic check")
		// StopWithReason has its own stopped guard so concurrent calls are safe
		go self.StopWithReason(context.Background(), true)
		return false
	}

	return true
}

// initTc initializes TC class and filter. Must be called with mu AND tcNftMu held.
func (self *RunningSession) initTc() error {
	classid := tc.GetAvailableId()
	defer classid.Cancel()

	net := self.network.Load()
	s := self.session

	err := net.lan.CreateClass(classid.Uint(), s.DownMbits(), s.UpMbits())
	if err != nil {
		return err
	}

	err = net.lan.CreateFilter(net.ip, classid.Uint())
	if err != nil {
		net.lan.DelClass(classid.Uint())
		return err
	}

	classid.Commit()
	self.tcClassId = &classid

	return nil
}

// updateTc updates TC class settings. Must be called with mu AND tcNftMu held.
func (self *RunningSession) updateTc() error {
	net := self.network.Load()
	s := self.session

	downMbits := s.DownMbits()
	upMbits := s.UpMbits()

	if s.UseGlobalSpeed() {
		d, u := net.lan.Bandwidth()
		downMbits, upMbits = int(d), int(u)
	}

	return net.lan.ChangeClass(self.tcClassId.Uint(), downMbits, upMbits)
}

// cleanUpTimer cleans up the timer. Must be called with mu held.
func (self *RunningSession) cleanUpTimer() {
	log.Println("Cleaning up session timer...")

	if self.timerCancel != nil {
		self.timerCancel() // Cancel the timer context
		self.timerCancel = nil
		self.timerCtx = nil
	}

	if self.timeTimer != nil {
		self.timeTimer.Stop()
		self.timeTimer = nil
	}

	log.Println("Done cleaning session timer.")
}

// snapshotTimeConsumption atomically bakes elapsed time into stored consumption and resets resumedAt.
// This is an internal bookkeeping operation - does NOT set dirty flags.
// Call persistSession() separately to save to DB.
// If clearResumed is true, sets ResumedAt to nil (session stopping).
// If clearResumed is false, resets ResumedAt to now (checkpoint for continued tracking).
// Returns elapsed seconds for logging purposes.
func (self *RunningSession) snapshotTimeConsumption(session sdkapi.IClientSession, clearResumed bool) int {
	if session == nil {
		return 0
	}
	return session.SnapshotTimeCons(clearResumed)
}

// persistSession saves session state directly to DB without triggering onSave callback.
// Used for internal bookkeeping operations (periodic saves, stop operations).
// Does NOT reload from DB - the in-memory state is already correct.
// Does NOT trigger the onSave callback - these are internal operations, not user-initiated changes.
func (self *RunningSession) persistSession(ctx context.Context, session sdkapi.IClientSession) error {
	if session == nil {
		return nil
	}
	return session.PersistToDB(ctx)
}

// ============================================================================
// SESSION UPDATE METHODS - Apply side effects after session.Save()
// These methods are called by SessionsMgr when a session is saved with changes.
// The session values are already updated; these methods apply runtime effects.
// ============================================================================

// ApplyTimeUpdateParams contains parameters for applying a time update.
type ApplyTimeUpdateParams struct {
	Ctx           context.Context
	RemainingSecs int
}

// ApplyTimeUpdate applies a time update to the running session.
// This resets the timer to the new remaining time.
// If remainingSecs is 0 and session type includes time, the session is stopped as consumed.
// Returns ErrSessionStopped if the session is already stopped.
// Note: Session values are already updated by the caller; this only applies runtime effects.
func (self *RunningSession) ApplyTimeUpdate(params ApplyTimeUpdateParams) error {
	self.mu.Lock()

	if self.stopped {
		self.mu.Unlock()
		return ErrSessionStopped
	}

	session := self.session
	if session == nil {
		self.mu.Unlock()
		return errors.New("no active session")
	}

	// Persist current consumption to the session before resetting timer
	// This ensures TimeConsumption() returns the stored value without elapsed calculation
	self.snapshotTimeConsumption(session, false)

	// Check if session should be stopped as consumed
	sessionType := session.Type()
	if params.RemainingSecs == 0 && (sessionType == sdkapi.SessionTypeTime || sessionType == sdkapi.SessionTypeTimeOrData) {
		self.mu.Unlock()
		return self.StopWithReason(params.Ctx, true)
	}

	// Reset the timer to the new remaining time
	self.resetTimer(params.RemainingSecs)

	self.mu.Unlock()
	return nil
}

// ApplyDataUpdate checks if the session should be stopped after a data update.
// If the session is consumed, it stops the session.
// Returns ErrSessionStopped if the session is already stopped.
// Note: Session values are already updated by the caller; this only checks consumption.
func (self *RunningSession) ApplyDataUpdate(ctx context.Context) error {
	self.mu.Lock()

	if self.stopped {
		self.mu.Unlock()
		return ErrSessionStopped
	}

	session := self.session
	if session == nil {
		self.mu.Unlock()
		return errors.New("no active session")
	}

	// Reset diffMb since the caller has already accounted for consumption
	self.diffMb = 0

	self.mu.Unlock()

	// Check if session is now consumed
	if session.IsConsumed() {
		return self.StopWithReason(ctx, true)
	}

	return nil
}

// ApplyBandwidthUpdateParams contains parameters for applying a bandwidth update.
type ApplyBandwidthUpdateParams struct {
	Ctx       context.Context
	DownMbits int
	UpMbits   int
	UseGlobal bool
}

// ApplyBandwidthUpdate applies a bandwidth update to the running session.
// This updates TC (traffic control) rules immediately and syncs the in-memory session.
// Returns ErrSessionStopped if the session is already stopped.
// Note: Session values are already saved to DB by the caller; this applies runtime effects.
func (self *RunningSession) ApplyBandwidthUpdate(params ApplyBandwidthUpdateParams) error {
	contextInfo := fmt.Sprintf("DeviceID=%d, Down=%d, Up=%d, UseGlobal=%v", self.clntId, params.DownMbits, params.UpMbits, params.UseGlobal)

	// Update TC rules
	return withTcNftLock("TC Bandwidth Update", contextInfo, func() error {
		self.mu.Lock()
		defer self.mu.Unlock()

		if self.stopped {
			return ErrSessionStopped
		}

		if self.session == nil {
			return errors.New("no active session")
		}

		// Update the in-memory session with the new bandwidth values.
		// This ensures self.session stays in sync with the database so that
		// future calls to updateTc() or initTc() use the correct values.
		self.session.SetData(sdkapi.SessionUpdateData{
			DownMbits:      &params.DownMbits,
			UpMbits:        &params.UpMbits,
			UseGlobalSpeed: &params.UseGlobal,
		})

		// Read network state inside lock to ensure consistency with UpdateNetworkDetails
		net := self.network.Load()

		// Calculate effective bandwidth - use global LAN bandwidth if UseGlobal is set
		downMbits := params.DownMbits
		upMbits := params.UpMbits
		if params.UseGlobal {
			d, u := net.lan.Bandwidth()
			downMbits, upMbits = int(d), int(u)
		}

		// Update TC class if it exists
		if self.tcClassId != nil {
			if err := net.lan.ChangeClass(self.tcClassId.Uint(), downMbits, upMbits); err != nil {
				return fmt.Errorf("failed to update TC class: %w", err)
			}
		}

		return nil
	})
}

// resetTimer cancels the existing timer and creates a new one with the specified duration.
// Must be called with mu held.
func (self *RunningSession) resetTimer(remainingSecs int) {
	// Clean up existing timer
	self.cleanUpTimer()

	// Start new timer
	self.startTimer(remainingSecs)
}
