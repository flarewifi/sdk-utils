package sessmgr

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"core/internal/modules/tc"
	"core/internal/network"
	jobque "core/utils/job-que"
	sdkapi "sdk/api"
)

// =============================================================================
// CONSTANTS AND VARIABLES
// =============================================================================

const (
	// bytesPerMiB is the number of bytes in a mebibyte (1024^2)
	bytesPerMiB = 1024 * 1024
)

var (
	ErrSessionExpired = errors.New("session expired")
	ErrSessionStopped = errors.New("session already stopped")
)

// =============================================================================
// TYPES
// =============================================================================

// RunningSession manages an active client session with bandwidth control.
//
// Field categories:
//   - IMMUTABLE: clnt, clntId, emitter, tcQueue - set at creation, never change, no lock needed
//   - ATOMIC: network, stopped - lock-free reads/writes via sync/atomic primitives
//   - MUTABLE: everything else - protected by mu
type RunningSession struct {
	// === IMMUTABLE after creation (no lock needed) ===
	clnt    sdkapi.IClientDevice
	clntId  int64
	emitter SessionEventEmitter
	tcQueue *jobque.JobQueue[struct{}] // reference to SessionsMgr's global TC serialization queue

	// === ATOMIC (lock-free) ===
	network atomic.Pointer[networkState]
	// stopped uses atomic.Bool so hot paths (UpdateDataConsumption, timerLoop,
	// handlePeriodicSave, IsStopped) can read it without acquiring mu.
	// StopWithReason uses CompareAndSwap to guarantee only one caller transitions
	// false→true (the one that "wins" proceeds; all others return immediately).
	stopped atomic.Bool

	// === MUTABLE STATE (protected by mu) ===
	mu            sync.Mutex
	tcClassId     *tc.TcClassId
	timeTimer     *time.Timer
	timerCancel   context.CancelFunc
	timerCtx      context.Context
	timerGen      uint64 // Generation counter to prevent stale timer goroutines from acting
	session       sdkapi.IClientSession
	diffMb        float64
	doneCh        chan error // Single buffered channel for session completion; reset by PrepareForChain()
	saveFailCount int        // Consecutive periodic save failures; reset on success
}

// =============================================================================
// CONSTRUCTOR
// =============================================================================

func NewRunningSession(clnt sdkapi.IClientDevice, s sdkapi.IClientSession, emitter SessionEventEmitter, tcQueue *jobque.JobQueue[struct{}]) (*RunningSession, error) {
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
		tcQueue: tcQueue,

		// Mutable state initialized
		session: s,
		doneCh:  make(chan error, 1),
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

// =============================================================================
// PUBLIC METHODS - Lock-free Accessors (immutable fields)
// =============================================================================

// ClientId returns the device ID (immutable, no lock needed).
func (self *RunningSession) ClientId() int64 {
	return self.clntId
}

// =============================================================================
// PUBLIC METHODS - Atomic Accessors (network state)
// =============================================================================

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

// =============================================================================
// PUBLIC METHODS - Mutex-Protected Accessors (mutable state)
// =============================================================================

// GetSession returns the current session (requires lock).
func (self *RunningSession) GetSession() sdkapi.IClientSession {
	self.mu.Lock()
	defer self.mu.Unlock()
	return self.session
}

// Done returns a channel that receives when the session ends.
// The channel is buffered (size 1) so StopWithReason never blocks.
// Call PrepareForChain() to reset the channel for the next session iteration.
func (self *RunningSession) Done() <-chan error {
	self.mu.Lock()
	defer self.mu.Unlock()
	return self.doneCh
}

// DiffMb returns the unsaved data consumption difference.
func (self *RunningSession) DiffMb() (mb float64) {
	self.mu.Lock()
	defer self.mu.Unlock()
	return self.diffMb
}

// IsStopped returns whether the session has been stopped (atomic, no lock needed).
func (self *RunningSession) IsStopped() bool {
	return self.stopped.Load()
}

// =============================================================================
// PUBLIC METHODS - Session Lifecycle
// =============================================================================

// PrepareForChain prepares the RunningSession for reuse with a new session.
// This enables session chaining: when one session expires, loopSessions calls
// PrepareForChain() then Start() with the next available session.
//
// Preserves: TC class/filter (avoids teardown/recreation overhead), network state, session reference.
// Resets: doneCh (new buffered channel), diffMb, saveFailCount.
//
// NOTE: stopped flag remains true until Start() successfully loads a new session.
// This ensures that if Stop() is called after PrepareForChain() but before Start()
// (e.g., when no more sessions are available), it's a no-op.
func (self *RunningSession) PrepareForChain() {
	self.mu.Lock()
	defer self.mu.Unlock()

	// Note: stopped remains true - only Start() clears it when new session loads
	self.doneCh = make(chan error, 1) // Fresh channel for next session
	self.diffMb = 0
	self.saveFailCount = 0
	// Note: session is intentionally preserved until Start() sets a new one
	// Note: tcClassId is intentionally preserved for reuse
	// Note: network state is intentionally preserved
}

func (self *RunningSession) Start(ctx context.Context, s sdkapi.IClientSession) error {
	net := self.network.Load()
	contextInfo := fmt.Sprintf("DeviceID=%d, SessionID=%d, MAC=%s, IP=%s",
		self.clntId, s.ID(), net.mac, net.ip)

	log.Printf("[Running Session] Start - %s", contextInfo)

	// 1. DB reload - no lock needed (session has its own synchronization)
	if err := s.Reload(ctx); err != nil {
		return fmt.Errorf("failed to reload session: %w", err)
	}

	// 2. Store session reference and clear stopped flag under mu.
	// Note: We allow Start() even if stopped=true (after PrepareForChain()) to enable session chaining.
	self.mu.Lock()
	self.stopped.Store(false)
	self.session = s
	self.mu.Unlock()

	// Build timestamp update outside mu — s.SetData() acquires ClientSession.writeMu
	// internally, so calling it while mu is held would create a mu→writeMu nesting.
	sessionData := s.Data()
	timeNow := time.Now().UTC()
	updateData := sdkapi.SessionUpdateData{}
	hasUpdates := false

	if sessionData.StartedAt == nil {
		updateData.StartedAt = &timeNow
		hasUpdates = true
	}
	if sessionData.ResumedAt == nil {
		updateData.ResumedAt = &timeNow
		hasUpdates = true
	}

	// Apply timestamp updates with no lock held (writeMu acquired internally).
	if hasUpdates {
		s.SetData(updateData)
	}

	// 3. Save to DB - no lock needed
	// Use PersistToDB() instead of Save() to avoid emitting EventSessionChanged.
	// EventSessionConnected is already emitted in loopSessions() when the session starts,
	// so we don't need a duplicate event here. This is an internal state transition (setting
	// started_at/resumed_at timestamps), not a user-initiated modification.
	if err := s.PersistToDB(ctx); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// 4. TC operations - check and init/update inside single lock acquisition.
	// Pass session explicitly so initTc/updateTc don't implicitly rely on self.session
	// being readable under mu — the contract is now explicit in the signature.
	err := self.execTc("TC Init/Update", func() error {
		self.mu.Lock()
		defer self.mu.Unlock()

		if self.tcClassId == nil {
			return self.initTc(s)
		}
		return self.updateTc(s)
	})
	if err != nil {
		return fmt.Errorf("failed to setup TC: %w", err)
	}

	// 5. Start timer if not already running.
	// initTimeTimer returns true when the session is already consumed and needs
	// an immediate stop. We dispatch StopWithReason AFTER releasing mu to avoid
	// calling it while mu is held (StopWithReason acquires mu internally via CAS).
	self.mu.Lock()
	stopImmediately := false
	if self.timeTimer == nil {
		stopImmediately = self.initTimeTimer(s)
		if !stopImmediately {
			log.Println("Session timer has started...")
		}
	}
	self.mu.Unlock()

	if stopImmediately {
		go self.StopWithReason(StopReasonConsumed)
	}

	log.Printf("[Running Session] Start completed - %s", contextInfo)
	return nil
}

func (self *RunningSession) Stop() error {
	return self.StopWithReason(StopReasonManual)
}

func (self *RunningSession) StopWithReason(reason StopReason) error {
	// Use CAS to guarantee only one caller transitions stopped false→true.
	// All concurrent callers that lose the race return immediately without
	// acquiring mu, eliminating contention on the hot "already stopped" path.
	if !self.stopped.CompareAndSwap(false, true) {
		log.Printf("[Running Session] StopWithReason - already stopped, DeviceID=%d", self.clntId)
		return nil
	}

	// Won the CAS — collect mutable state and clean up timer under mu.
	self.mu.Lock()
	session := self.session
	doneCh := self.doneCh
	self.cleanUpTimer()
	self.mu.Unlock()

	// Build context info for logging
	var sessionID int64
	if session != nil {
		sessionID = session.ID()
	}
	contextInfo := fmt.Sprintf("DeviceID=%d, SessionID=%d, Reason=%s",
		self.clntId, sessionID, reason)

	log.Printf("[Running Session] StopWithReason - %s", contextInfo)

	// Calculate and record elapsed time
	elapsed := self.snapshotTimeConsumption(session, true)
	if elapsed > 0 && session != nil {
		log.Printf("Recording elapsed time: %d seconds (total consumption: %d)\n",
			elapsed, session.TimeConsumption())
	}

	// DB save - no lock needed (session has its own synchronization)
	saveErr := self.persistSession(session)

	// Emit events - only emit if we have a valid session.
	// Session can be nil if Stop() is called before Start() completes
	// or if the session was never properly initialized.
	// Note: emitter and clnt are immutable fields set at construction.
	if session != nil {
		if reason == StopReasonConsumed {
			// Session was consumed (time/data exhausted) - emit both consumed and disconnected
			self.emitter.EmitSessionEvent(sdkapi.EventSessionConsumed, sdkapi.SessionEventData{Session: session})
		}
		// Always emit disconnected event when session stops
		self.emitter.EmitSessionEvent(sdkapi.EventSessionDisconnected, sdkapi.SessionEventData{Session: session})
	}

	// Determine the error to signal to Done() waiters.
	// For consumed sessions, always return ErrSessionExpired but include saveErr context if present.
	// For manual stops, propagate saveErr directly (nil on success).
	var callbackErr error
	if reason == StopReasonConsumed {
		if saveErr != nil {
			// Wrap both errors - session expired AND save failed.
			// Data loss is mitigated by periodic saves (at most 1 minute lost).
			callbackErr = fmt.Errorf("%w (save failed: %v)", ErrSessionExpired, saveErr)
			log.Printf("[Running Session] WARNING - Save failed during consumed stop: %v", saveErr)
		} else {
			callbackErr = ErrSessionExpired
		}
	} else {
		callbackErr = saveErr
	}

	// Signal completion to Done() waiter - non-blocking send since channel is buffered
	select {
	case doneCh <- callbackErr:
	default:
		// This should never happen with a properly managed buffered channel.
		// If it does, it indicates a bug (double-stop or channel misuse).
		log.Printf("[Running Session] WARNING - doneCh full, could not signal completion - DeviceID=%d", self.clntId)
	}
	log.Println("Session stop signaled.")

	log.Printf("[Running Session] StopWithReason completed - %s", contextInfo)
	return callbackErr
}

func (self *RunningSession) CleanupTc() error {
	return self.execTc("CleanupTc", func() error {
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

// =============================================================================
// PUBLIC METHODS - Network Update
// =============================================================================

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

	// All TC operations and network state updates happen inside execTc+mu.
	// Re-read network state inside the lock to handle concurrent UpdateNetworkDetails calls.
	// If another call already updated the state, we detect it and adjust accordingly.
	return self.execTc("TC Network Update", func() error {
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
			if err := self.initTc(self.session); err != nil {
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

// =============================================================================
// PUBLIC METHODS - Data Consumption
// =============================================================================

func (self *RunningSession) UpdateDataConsumption(stats *sdkapi.TrafficData) {
	// Fast atomic check — no lock needed for stopped.
	if self.stopped.Load() {
		return
	}

	// Grab session reference under lock (session pointer is mu-protected).
	self.mu.Lock()
	session := self.session
	self.mu.Unlock()

	if session == nil {
		return
	}

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
	// Check stopped again (atomic) — if StopWithReason won the CAS between
	// the first check and here, we skip the diffMb update and shouldStop path.
	if self.stopped.Load() {
		return
	}
	self.mu.Lock()
	self.diffMb += dataconMb
	shouldStop := session.IsConsumed()
	self.mu.Unlock()

	if shouldStop {
		log.Println("Session data is consumed!!!")
		// StopWithReason has its own stopped guard, so concurrent calls are safe (only first wins)
		go self.StopWithReason(StopReasonConsumed)
	}
}

// =============================================================================
// PUBLIC METHODS - Session Update (side effects after session.Save())
// =============================================================================

// ApplyTimeUpdate applies a time update to the running session.
// This resets the timer to the new remaining time.
// If remainingSecs is 0 and session type includes time, the session is stopped as consumed.
// Returns ErrSessionStopped if the session is already stopped.
// Note: Session values are already updated by the caller; this only applies runtime effects.
func (self *RunningSession) ApplyTimeUpdate(params ApplyTimeUpdateParams) error {
	// Phase 1: capture state under lock, then release before calling session methods.
	// snapshotTimeConsumption acquires ClientSession.writeMu internally; holding RunningSession.mu
	// at the same time creates a nested-lock ordering risk (mu → writeMu). Release mu first
	// to eliminate the nesting entirely.
	if self.stopped.Load() {
		return ErrSessionStopped
	}

	self.mu.Lock()
	session := self.session
	self.mu.Unlock()

	if session == nil {
		return errors.New("no active session")
	}

	sessionType := session.Type()

	// Phase 2: persist current consumption — no lock held (writeMu acquired internally).
	// This ensures TimeConsumption() returns the stored value without elapsed calculation.
	self.snapshotTimeConsumption(session, false)

	// Check if session should be stopped as consumed
	if params.RemainingSecs == 0 && (sessionType == sdkapi.SessionTypeTime || sessionType == sdkapi.SessionTypeTimeOrData) {
		return self.StopWithReason(StopReasonConsumed)
	}

	// Phase 3: re-acquire lock only for the timer reset (pure RunningSession state).
	if self.stopped.Load() {
		return ErrSessionStopped
	}
	self.mu.Lock()
	self.resetTimer(params.RemainingSecs)
	self.mu.Unlock()

	return nil
}

// ApplyDataUpdate checks if the session should be stopped after a data update.
// If the session is consumed, it stops the session.
// Returns ErrSessionStopped if the session is already stopped.
// Note: Session values are already updated by the caller; this only checks consumption.
func (self *RunningSession) ApplyDataUpdate() error {
	if self.stopped.Load() {
		return ErrSessionStopped
	}

	self.mu.Lock()
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
		return self.StopWithReason(StopReasonConsumed)
	}

	return nil
}

// ApplyBandwidthUpdate applies a bandwidth update to the running session.
// This updates TC (traffic control) rules immediately and syncs the in-memory session.
// Returns ErrSessionStopped if the session is already stopped.
// Note: Session values are already saved to DB by the caller; this applies runtime effects.
func (self *RunningSession) ApplyBandwidthUpdate(params ApplyBandwidthUpdateParams) error {
	// Phase 1: capture session reference under lock, then release before calling SetData.
	// session.SetData() acquires ClientSession.writeMu internally; holding RunningSession.mu
	// at the same time creates a nested-lock ordering risk (mu → writeMu). Release mu first.
	if self.stopped.Load() {
		return ErrSessionStopped
	}
	self.mu.Lock()
	session := self.session
	self.mu.Unlock()

	if session == nil {
		return errors.New("no active session")
	}

	// Phase 2: update in-memory session — no lock held (writeMu acquired internally).
	// This ensures self.session stays in sync with the database so that
	// future calls to updateTc() or initTc() use the correct values.
	session.SetData(sdkapi.SessionUpdateData{
		DownMbits:      &params.DownMbits,
		UpMbits:        &params.UpMbits,
		UseGlobalSpeed: &params.UseGlobal,
	})

	// Phase 3: update TC rules — re-acquires mu inside execTc callback.
	return self.execTc("TC Bandwidth Update", func() error {
		self.mu.Lock()
		defer self.mu.Unlock()

		if self.stopped.Load() {
			return ErrSessionStopped
		}

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

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// execTc submits a TC/NFT operation to the global serialization queue and waits for the result.
// This ensures all TC/NFT commands execute one at a time across all sessions.
// Uses JobQueue for automatic panic recovery, queue-wait logging, and context support.
func (self *RunningSession) execTc(operation string, fn func() error) error {
	contextInfo := fmt.Sprintf("DeviceID=%d", self.clntId)
	_, err := self.tcQueue.ExecWithContext(
		context.Background(),
		operation,
		contextInfo,
		func() (struct{}, error) {
			return struct{}{}, fn()
		},
	)
	return err
}

// initTimeTimer initializes the session timer. Must be called with mu held.
// Returns true if the session is already consumed and the caller should trigger
// an immediate stop AFTER releasing mu (to avoid calling StopWithReason while mu is held).
func (self *RunningSession) initTimeTimer(s sdkapi.IClientSession) bool {
	// Get session data in a single atomic snapshot
	data := s.Data()

	// If session is already consumed, signal the caller to stop after releasing mu.
	// We never dispatch goroutines while holding mu — that is fragile and hard to reason about.
	if data.IsConsumed {
		log.Println("Session already consumed or expired, stopping immediately")
		return true
	}

	// Start the timer for the remaining time
	self.startTimer(data.RemainingTime)
	return false
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
	// Add a random jitter of 0–15 seconds before the first tick so that all
	// active sessions don't write to the database simultaneously every minute.
	// This spreads the write load and reduces "database is locked" contention.
	jitter := time.Duration(rand.Int63n(int64(15 * time.Second)))
	select {
	case <-ctx.Done():
		return
	case <-time.After(jitter):
	}

	saveTicker := time.NewTicker(1 * time.Minute)
	defer saveTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Session timer cancelled")
			return

		case <-timer.C:
			// Verify this timer goroutine is still current before acting.
			// stopped is atomic — read it without mu. timerGen still needs mu.
			self.mu.Lock()
			currentGen := self.timerGen
			self.mu.Unlock()
			if currentGen != gen || self.stopped.Load() {
				log.Printf("[Running Session] Stale timer expired (gen=%d, current=%d), ignoring", gen, currentGen)
				return
			}

			log.Println("Session timer expired - time consumed!")
			self.StopWithReason(StopReasonConsumed)
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
	// stopped is atomic — check it before acquiring mu to short-circuit fast.
	if self.stopped.Load() {
		return false
	}
	self.mu.Lock()
	if self.timerGen != gen {
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
	self.snapshotTimeConsumption(currentSession, false)
	log.Printf("Periodic save: persisting %d seconds consumed, %d remaining\n",
		currentSession.ConsumedTimeSecs(), currentSession.RemainingTime())

	saveErr := self.persistSession(currentSession)

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
			go self.Stop()
			return false
		}
		log.Printf("[WARN] Periodic save failed (%d/%d): %v - will retry next period", failCount, maxConsecutiveFailures, saveErr)
		return true
	}

	// Save succeeded - reset failure counter and check for stale/consumed state.
	// stopped is atomic — read it outside mu; timerGen still needs mu.
	self.mu.Lock()
	self.saveFailCount = 0
	freshSession := self.session
	staleGen := self.timerGen != gen
	self.mu.Unlock()

	if self.stopped.Load() || staleGen {
		return false
	}

	if freshSession != nil && freshSession.IsConsumed() {
		log.Println("Session consumed or expired during periodic check")
		// StopWithReason has its own stopped guard so concurrent calls are safe
		go self.StopWithReason(StopReasonConsumed)
		return false
	}

	return true
}

// initTc initializes TC class and filter. Must be called from within execTc with mu held.
// Session is passed explicitly rather than read from self.session to make the
// dependency on mu-protected state visible at the call site.
func (self *RunningSession) initTc(s sdkapi.IClientSession) error {
	classid := tc.GetAvailableId()
	defer classid.Cancel()

	net := self.network.Load()

	// Get session data in a single atomic snapshot
	data := s.Data()

	err := net.lan.CreateClass(classid.Uint(), data.DownMbits, data.UpMbits)
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

// updateTc updates TC class settings. Must be called from within execTc with mu held.
// Session is passed explicitly rather than read from self.session to make the
// dependency on mu-protected state visible at the call site.
func (self *RunningSession) updateTc(s sdkapi.IClientSession) error {
	net := self.network.Load()

	// Get session data in a single atomic snapshot
	data := s.Data()

	downMbits := data.DownMbits
	upMbits := data.UpMbits

	if data.UseGlobalSpeed {
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
func (self *RunningSession) persistSession(session sdkapi.IClientSession) error {
	if session == nil {
		return nil
	}
	return session.PersistToDB(context.Background())
}

// resetTimer cancels the existing timer and creates a new one with the specified duration.
// Must be called with mu held.
func (self *RunningSession) resetTimer(remainingSecs int) {
	// Clean up existing timer
	self.cleanUpTimer()

	// Start new timer
	self.startTimer(remainingSecs)
}
