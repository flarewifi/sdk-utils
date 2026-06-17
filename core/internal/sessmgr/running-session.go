package sessmgr

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"core/internal/modules/tc"
	"core/internal/modules/validation"
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
	// IsStopped) can read it without acquiring mu.
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
	initNet := &networkState{ipv4: clnt.Ipv4Addr(), ipv6: clnt.Ipv6Addr()}
	primaryAddr := primaryIP(initNet)
	lan, err := network.FindByIp(primaryAddr)
	if err != nil {
		return nil, err
	}

	rs := &RunningSession{
		clnt:    clnt,
		clntId:  clnt.ID(),
		emitter: emitter,
		tcQueue: tcQueue,

		session: s,
		doneCh:  make(chan error, 1),
	}

	rs.network.Store(&networkState{
		ipv4: clnt.Ipv4Addr(),
		ipv6: clnt.Ipv6Addr(),
		mac:  clnt.MacAddr(),
		lan:  lan,
	})

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

// IpAddr returns the primary IP address (IPv4 if available, else IPv6).
// Atomic read, no lock needed.
func (self *RunningSession) IpAddr() string {
	n := self.network.Load()
	if n.ipv4 != "" {
		return n.ipv4
	}
	return n.ipv6
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
	// 1. DB reload - no lock needed (session has its own synchronization)
	if err := s.Reload(ctx); err != nil {
		return fmt.Errorf("failed to reload session: %w", err)
	}

	// 2. Validate session fields against freshly reloaded data before touching DB or TC.
	sessionData := s.Data()
	if err := validation.ValidateSessionData(sessionData.Type, sessionData.DownMbits, sessionData.UpMbits, sessionData.UseGlobalSpeed); err != nil {
		return fmt.Errorf("invalid session %d: %w", sessionData.ID, err)
	}

	// 3. Store session reference and clear stopped flag under mu.
	// Note: We allow Start() even if stopped=true (after PrepareForChain()) to enable session chaining.
	self.mu.Lock()
	self.stopped.Store(false)
	self.session = s
	self.mu.Unlock()

	// Build timestamp update outside mu — s.SetData() acquires ClientSession.writeMu
	// internally, so calling it while mu is held would create a mu→writeMu nesting.
	// sessionData was already fetched above for validation; reuse it here.
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
	}
	self.mu.Unlock()

	if stopImmediately {
		go self.StopWithReason(StopReasonConsumed)
	}

	return nil
}

func (self *RunningSession) Stop() error {
	return self.StopWithReason(StopReasonManual)
}

func (self *RunningSession) StopWithReason(reason StopReason) error {
	if !self.stopped.CompareAndSwap(false, true) {
		return nil
	}

	self.mu.Lock()
	session := self.session
	doneCh := self.doneCh
	self.cleanUpTimer()
	self.mu.Unlock()

	saveErr := self.persistSession(session)

	if session != nil {
		if reason == StopReasonConsumed {
			self.emitter.EmitSessionEvent(context.Background(), sdkapi.EventSessionConsumed, sdkapi.SessionEventData{Session: session})
		}
		self.emitter.EmitSessionEvent(context.Background(), sdkapi.EventSessionDisconnected, sdkapi.SessionEventData{Session: session})
	}

	var callbackErr error
	if reason == StopReasonConsumed {
		if saveErr != nil {
			callbackErr = fmt.Errorf("%w (save failed: %v)", ErrSessionExpired, saveErr)
		} else {
			callbackErr = ErrSessionExpired
		}
	} else {
		callbackErr = saveErr
	}

	select {
	case doneCh <- callbackErr:
	default:
	}

	return callbackErr
}

func (self *RunningSession) CleanupTc() error {
	return self.execTc("CleanupTc", func() error {
		self.mu.Lock()
		defer self.mu.Unlock()

		net := self.network.Load()

		if self.tcClassId != nil {
			classid := *self.tcClassId

			if net.ipv4 != "" {
				net.lan.DelFilter(net.ipv4, classid.Uint())
			}
			if net.ipv6 != "" {
				net.lan.DelFilter(net.ipv6, classid.Uint())
			}

			if err := net.lan.DelClass(classid.Uint()); err != nil {
				return err
			}

			self.tcClassId = nil
		}

		return nil
	})
}

// =============================================================================
// PUBLIC METHODS - Network Update
// =============================================================================

// UpdateNetworkDetails updates the MAC, IPv4, and IPv6 addresses when device network details change.
func (self *RunningSession) UpdateNetworkDetails(ctx context.Context, newMac, newIpv4, newIpv6 string) error {
	quickNet := self.network.Load()
	if quickNet.ipv4 == newIpv4 && quickNet.ipv6 == newIpv6 && quickNet.mac == newMac {
		return nil
	}

	newPrimaryAddr := primaryIP(&networkState{ipv4: newIpv4, ipv6: newIpv6})
	quickPrimaryAddr := primaryIP(quickNet)

	var newLan *network.NetworkLan
	if quickPrimaryAddr != newPrimaryAddr {
		var err error
		newLan, err = network.FindByIp(newPrimaryAddr)
		if err != nil {
			return err
		}
	}

	return self.execTc("TC Network Update", func() error {
		self.mu.Lock()
		defer self.mu.Unlock()

		currentNet := self.network.Load()

		if currentNet.ipv4 == newIpv4 && currentNet.ipv6 == newIpv6 && currentNet.mac == newMac {
			return nil
		}

		if currentNet.ipv4 == newIpv4 && currentNet.ipv6 == newIpv6 {
			self.network.Store(&networkState{
				ipv4: newIpv4,
				ipv6: newIpv6,
				mac:  newMac,
				lan:  currentNet.lan,
			})
			return nil
		}

		if newLan == nil {
			newLan = currentNet.lan
		}

		if newLan.Name() != currentNet.lan.Name() {
			if self.tcClassId != nil {
				classid := self.tcClassId.Uint()
				if currentNet.ipv4 != "" {
					currentNet.lan.DelFilter(currentNet.ipv4, classid)
				}
				if currentNet.ipv6 != "" {
					currentNet.lan.DelFilter(currentNet.ipv6, classid)
				}

				currentNet.lan.DelClass(classid)
				self.tcClassId = nil
			}

			self.network.Store(&networkState{
				ipv4: newIpv4,
				ipv6: newIpv6,
				mac:  newMac,
				lan:  newLan,
			})

			if err := self.initTc(self.session); err != nil {
				return err
			}
		} else {
			if self.tcClassId != nil {
				classid := self.tcClassId.Uint()

				if currentNet.ipv4 != "" {
					currentNet.lan.DelFilter(currentNet.ipv4, classid)
				}
				if currentNet.ipv6 != "" {
					currentNet.lan.DelFilter(currentNet.ipv6, classid)
				}

				if newIpv4 != "" {
					if err := newLan.CreateFilter(newIpv4, classid); err != nil {
						return err
					}
				}
				if newIpv6 != "" {
					if err := newLan.CreateFilter(newIpv6, classid); err != nil {
						return err
					}
				}
			}

			self.network.Store(&networkState{
				ipv4: newIpv4,
				ipv6: newIpv6,
				mac:  newMac,
				lan:  newLan,
			})
		}

		return nil
	})
}

// primaryIP returns the primary IP from a networkState (IPv4 preferred, fallback IPv6).
func primaryIP(net *networkState) string {
	if net.ipv4 != "" {
		return net.ipv4
	}
	return net.ipv6
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

	// Both Download and Upload are now keyed by MAC address.
	// Only record consumption if at least one direction has data this tick;
	// requiring both risks discarding valid partial ticks (e.g. download-only).
	macUpper := strings.ToUpper(net.mac)
	dl, dlOK := stats.Download[macUpper]
	upload, upOK := stats.Upload[macUpper]

	if !dlOK && !upOK {
		return
	}

	dataconMb := float64(dl.Bytes+upload.Bytes) / bytesPerMiB

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

	if data.IsConsumed {
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

	go self.timerLoop(ctx, timer, gen)
}

// timerLoop handles timer expiration (session consumed).
// Runs in a separate goroutine. The gen parameter ensures stale goroutines
// (from a previous timer that was cancelled and replaced) become no-ops.
// Periodic snapshots and DB persistence are handled by SessionsMgr's batch
// save loop — this goroutine only watches for time expiration.
func (self *RunningSession) timerLoop(ctx context.Context, timer *time.Timer, gen uint64) {
	for {
		select {
		case <-ctx.Done():
			return

		case <-timer.C:
			self.mu.Lock()
			currentGen := self.timerGen
			self.mu.Unlock()
			if currentGen != gen || self.stopped.Load() {
				return
			}

			self.StopWithReason(StopReasonConsumed)
			return
		}
	}
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

	// Create filter for IPv4 (primary, required if present)
	if net.ipv4 != "" {
		err = net.lan.CreateFilter(net.ipv4, classid.Uint())
		if err != nil {
			net.lan.DelClass(classid.Uint())
			return err
		}
	}

	// Create filter for IPv6 (required if present — silent failure would let IPv6 traffic bypass bandwidth quota).
	if net.ipv6 != "" {
		if err := net.lan.CreateFilter(net.ipv6, classid.Uint()); err != nil {
			// Roll back: remove IPv4 filter and class before returning error.
			if net.ipv4 != "" {
				net.lan.DelFilter(net.ipv4, classid.Uint())
			}
			net.lan.DelClass(classid.Uint())
			return fmt.Errorf("failed to create IPv6 TC filter: %w", err)
		}
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
	if self.timerCancel != nil {
		self.timerCancel()
		self.timerCancel = nil
		self.timerCtx = nil
	}

	if self.timeTimer != nil {
		self.timeTimer.Stop()
		self.timeTimer = nil
	}
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

// incrementSaveFailCount increments the consecutive save failure counter.
// If the counter reaches the maximum, the session is stopped.
// Called by SessionsMgr's batch save loop when a flush fails.
func (self *RunningSession) incrementSaveFailCount() {
	self.mu.Lock()
	self.saveFailCount++
	count := self.saveFailCount
	self.mu.Unlock()

	const maxConsecutiveFailures = 3
	if count >= maxConsecutiveFailures {
		go self.Stop()
	}
}

// resetSaveFailCount resets the consecutive save failure counter to zero.
// Called by SessionsMgr's batch save loop after a successful flush.
func (self *RunningSession) resetSaveFailCount() {
	self.mu.Lock()
	self.saveFailCount = 0
	self.mu.Unlock()
}

// resetTimer cancels the existing timer and creates a new one with the specified duration.
// Must be called with mu held.
func (self *RunningSession) resetTimer(remainingSecs int) {
	// Clean up existing timer
	self.cleanUpTimer()

	// Start new timer
	self.startTimer(remainingSecs)
}
