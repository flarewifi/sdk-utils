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
	mu          sync.Mutex // Changed from RWMutex - simpler, less error-prone
	tcClassId   *tc.TcClassId
	timeTimer   *time.Timer
	timerCancel context.CancelFunc
	timerCtx    context.Context
	session     sdkapi.IClientSession
	diffMb      float64
	callbacks   []chan error
	stopped     bool // Prevents operations after Stop()
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

// ============================================================================
// NETWORK UPDATE - Atomic swap with TC operations
// ============================================================================

// UpdateNetworkDetails updates the MAC and IP address when device network details change.
func (self *RunningSession) UpdateNetworkDetails(ctx context.Context, newMac, newIP string) error {
	oldNet := self.network.Load()
	contextInfo := fmt.Sprintf("DeviceID=%d, OldMAC=%s, NewMAC=%s, OldIP=%s, NewIP=%s",
		self.clntId, oldNet.mac, newMac, oldNet.ip, newIP)

	log.Printf("[Running Session] UpdateNetworkDetails - %s", contextInfo)

	// Check if network details actually changed
	if oldNet.ip == newIP && oldNet.mac == newMac {
		log.Printf("[Running Session] No network changes detected for device %d", self.clntId)
		return nil
	}

	// Determine new LAN (might be different network)
	newLan := oldNet.lan
	if oldNet.ip != newIP {
		log.Printf("[Running Session] IP changed, checking if LAN changed...")
		var err error
		newLan, err = network.FindByIp(newIP)
		if err != nil {
			log.Printf("[Running Session] ERROR - Failed to find LAN for new IP %s: %v", newIP, err)
			return err
		}
	}

	// Handle TC rule updates if IP changed
	// NOTE: Network state is updated INSIDE the TC lock to ensure readers never see
	// a network state without matching TC rules (prevents race condition where
	// UpdateDataConsumption could use new IP before TC filter exists)
	if oldNet.ip != newIP {
		if newLan.Name() != oldNet.lan.Name() {
			// LAN changed - need to recreate TC rules on new interface
			log.Printf("[Running Session] LAN changed from %s to %s, recreating TC rules...",
				oldNet.lan.Name(), newLan.Name())

			err := withTcNftLock("TC Network Update (LAN changed)", contextInfo, func() error {
				self.mu.Lock()
				defer self.mu.Unlock()

				// Clean up old TC rules
				if self.tcClassId != nil {
					classid := self.tcClassId.Uint()
					log.Printf("[Running Session] Removing old TC filter for IP %s", oldNet.ip)
					if err := oldNet.lan.DelFilter(oldNet.ip, classid); err != nil {
						log.Printf("[Running Session] WARNING - Failed to delete old filter: %v", err)
					}

					log.Printf("[Running Session] Removing old TC class %d", classid)
					if err := oldNet.lan.DelClass(classid); err != nil {
						log.Printf("[Running Session] WARNING - Failed to delete old class: %v", err)
					}
					self.tcClassId = nil
				}

				// Recreate TC rules on new interface
				log.Printf("[Running Session] Creating new TC rules on interface %s", newLan.Name())
				if err := self.initTc(); err != nil {
					log.Printf("[Running Session] ERROR - Failed to create TC rules: %v", err)
					return err
				}
				log.Printf("[Running Session] TC rules recreated successfully")

				// Update network state AFTER TC rules are ready
				// Readers will now see the new state with TC rules already in place
				self.network.Store(&networkState{
					ip:  newIP,
					mac: newMac,
					lan: newLan,
				})

				return nil
			})
			if err != nil {
				return err
			}
		} else {
			// Same LAN, just update the filter
			log.Printf("[Running Session] Same LAN, updating TC filter from IP %s to %s", oldNet.ip, newIP)

			err := withTcNftLock("TC Network Update (same LAN)", contextInfo, func() error {
				self.mu.Lock()
				defer self.mu.Unlock()

				if self.tcClassId != nil {
					classid := self.tcClassId.Uint()

					// Remove old filter
					if err := oldNet.lan.DelFilter(oldNet.ip, classid); err != nil {
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
				// Readers will now see the new state with TC rules already in place
				self.network.Store(&networkState{
					ip:  newIP,
					mac: newMac,
					lan: newLan,
				})

				return nil
			})
			if err != nil {
				return err
			}
		}
	} else if oldNet.mac != newMac {
		// Only MAC changed (IP same) - no TC rules to update, just update network state
		self.network.Store(&networkState{
			ip:  newIP,
			mac: newMac,
			lan: newLan,
		})
	}

	log.Printf("[Running Session] Network details updated successfully - DeviceID=%d, NewMAC=%s, NewIP=%s",
		self.clntId, newMac, newIP)
	return nil
}

// ============================================================================
// SESSION LIFECYCLE
// ============================================================================

func (self *RunningSession) Start(ctx context.Context, s sdkapi.IClientSession) error {
	net := self.network.Load()
	contextInfo := fmt.Sprintf("DeviceID=%d, SessionID=%d, MAC=%s, IP=%s",
		self.clntId, s.ID(), net.mac, net.ip)

	log.Printf("[Running Session] Start - %s", contextInfo)

	// Check if already stopped
	self.mu.Lock()
	if self.stopped {
		self.mu.Unlock()
		return ErrSessionStopped
	}
	self.mu.Unlock()

	// 1. DB operations - no TC/NFT lock needed
	dbStart := time.Now()
	if err := s.Reload(ctx); err != nil {
		return fmt.Errorf("failed to reload session: %w", err)
	}
	logSlowOperation("DB Reload", dbStart, 2*time.Second, contextInfo)

	// 2. Update session state
	self.mu.Lock()
	self.session = s

	// Set first start time if this is the first time session is starting
	timeNow := time.Now().UTC()
	if s.StartedAt() == nil {
		s.SetStartedAt(&timeNow)
	}

	// Set resumed time to track current running period
	if s.ResumedAt() == nil {
		s.SetResumedAt(&timeNow)
	}
	self.mu.Unlock()

	// 3. Save to DB - no TC/NFT lock needed
	dbStart = time.Now()
	if err := s.Save(ctx); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	logSlowOperation("DB Save", dbStart, 2*time.Second, contextInfo)

	// 4. TC operations - use global tcNftMu lock
	self.mu.Lock()
	hasTcClass := self.tcClassId != nil
	self.mu.Unlock()

	if !hasTcClass {
		err := withTcNftLock("TC Init", contextInfo, func() error {
			self.mu.Lock()
			defer self.mu.Unlock()
			return self.initTc()
		})
		if err != nil {
			return fmt.Errorf("failed to init TC: %w", err)
		}
	} else {
		err := withTcNftLock("TC Update", contextInfo, func() error {
			self.mu.Lock()
			defer self.mu.Unlock()
			return self.updateTc()
		})
		if err != nil {
			return fmt.Errorf("failed to update TC: %w", err)
		}
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

func (self *RunningSession) StopWithReason(ctx context.Context, expired bool) error {
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
		self.clntId, sessionID, expired)

	log.Printf("[Running Session] StopWithReason - %s", contextInfo)

	// Calculate and record elapsed time
	if session != nil && session.ResumedAt() != nil {
		// TimeConsumption() already includes elapsed time since resumed_at
		currentCons := session.TimeConsumption()
		session.SetTimeCons(currentCons)

		// Calculate elapsed for logging only
		elapsed := int(time.Since(*session.ResumedAt()).Seconds())
		log.Printf("Recording elapsed time: %d seconds (total consumption: %d)\n",
			elapsed, currentCons)

		// Reset resumed_at to nil since session is stopping
		session.SetResumedAt(nil)
	}

	// DB save - no lock needed (session has its own synchronization)
	dbStart := time.Now()
	saveErr := self.saveSession(ctx, session)
	logSlowOperation("DB Save (Stop)", dbStart, 2*time.Second, contextInfo)

	// Emit events (emitter is immutable, no lock needed)
	if expired && self.emitter != nil && self.clnt != nil {
		self.emitter.emitSessionEvent(sdkapi.EventSessionBeforeExpired, session, self.clnt)
		self.emitter.emitSessionEvent(sdkapi.EventSessionExpired, session, self.clnt)
	}

	// Determine the error to return to callbacks
	var callbackErr error
	if expired {
		callbackErr = ErrSessionExpired
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
	net := self.network.Load()
	contextInfo := fmt.Sprintf("DeviceID=%d, IP=%s", self.clntId, net.ip)

	return withTcNftLock("CleanupTc", contextInfo, func() error {
		self.mu.Lock()
		defer self.mu.Unlock()

		if self.tcClassId != nil {
			log.Println("Clean up TC...")
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
	// Check if stopped first
	self.mu.Lock()
	if self.stopped {
		self.mu.Unlock()
		return
	}
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

	// Update diffMb and check if consumed
	self.mu.Lock()
	self.diffMb += dataconMb
	shouldStop := self.isConsumed()
	self.mu.Unlock()

	if shouldStop {
		log.Println("Session data is consumed!!!")
		go self.StopWithReason(context.Background(), true)
	}
}

// ============================================================================
// INTERNAL HELPERS - Must be called with appropriate locks held
// ============================================================================

func (self *RunningSession) initTimeTimer(s sdkapi.IClientSession) {
	// Calculate remaining time
	remainingSecs := s.RemainingTime()

	if remainingSecs <= 0 {
		log.Println("Session time already consumed, stopping immediately")
		go self.StopWithReason(context.Background(), true)
		return
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	self.timerCtx = ctx
	self.timerCancel = cancel

	// Create timer for remaining duration
	duration := time.Duration(remainingSecs) * time.Second
	timer := time.NewTimer(duration)
	self.timeTimer = timer

	log.Printf("Session timer started for %d seconds\n", remainingSecs)

	// Start timer goroutine
	go func() {
		// Periodic save ticker (every 1 minute)
		saveTicker := time.NewTicker(1 * time.Minute)
		defer saveTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Timer was cancelled
				log.Println("Session timer cancelled")
				return

			case <-timer.C:
				// Timer expired - session time consumed
				log.Println("Session timer expired - time consumed!")
				go self.StopWithReason(context.Background(), true)
				return

			case <-saveTicker.C:
				// Check if stopped
				self.mu.Lock()
				if self.stopped {
					self.mu.Unlock()
					return
				}
				currentSession := self.session
				self.mu.Unlock()

				if currentSession == nil {
					continue
				}

				// Persist time consumption to protect against crashes
				// This ensures at most 15 seconds of time tracking is lost on crash
				// instead of all time since session start
				var elapsed int
				if resumedAt := currentSession.ResumedAt(); resumedAt != nil {
					// Get current total consumption (includes elapsed since resumed_at)
					currentCons := currentSession.TimeConsumption()
					elapsed = int(time.Since(*resumedAt).Seconds())

					// Update timeCons with the accumulated time
					currentSession.SetTimeCons(currentCons)

					// Reset resumed_at to NOW so next calculation starts fresh
					// This prevents double-counting: timeCons now includes elapsed,
					// and RemainingTime() will calculate new elapsed from this point
					now := time.Now().UTC()
					currentSession.SetResumedAt(&now)

					log.Printf("Periodic save: persisting %d seconds consumed, %d remaining\n",
						currentCons, currentSession.RemainingTime())
				}

				// Emit before updated hook (emitter is immutable)
				if self.emitter != nil && self.clnt != nil {
					if err := self.emitter.emitSessionEvent(sdkapi.EventSessionBeforeUpdated, currentSession, self.clnt); err != nil {
						log.Println("Before update hook failed:", err)
						go self.Stop(context.Background())
						return
					}
				}

				// Direct save
				contextInfo := fmt.Sprintf("DeviceID=%d, SessionID=%d, Elapsed=%ds",
					self.clntId, currentSession.ID(), elapsed)

				dbStart := time.Now()
				saveErr := self.saveSession(context.Background(), currentSession)
				logSlowOperation("Periodic Save", dbStart, 2*time.Second, contextInfo)

				if saveErr != nil {
					log.Printf("[ERROR] Periodic save failed: %v - STOPPING SESSION", saveErr)
					go self.Stop(context.Background())
					return
				}

				// Emit session:updated event after successful save
				if self.emitter != nil && self.clnt != nil {
					self.emitter.emitSessionEvent(sdkapi.EventSessionUpdated, currentSession, self.clnt)
				}

				// Reset diff counter for data
				self.mu.Lock()
				self.diffMb = 0
				self.mu.Unlock()
			}
		}
	}()
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
	useGlobal := s.UseGlobalSpeed()

	if useGlobal {
		lan, err := network.FindByIp(net.ip)
		if err != nil {
			return err
		}

		d, u := lan.Bandwidth()
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

// saveSession saves and reloads a session.
func (self *RunningSession) saveSession(ctx context.Context, session sdkapi.IClientSession) error {
	if session == nil {
		return nil
	}

	if err := session.Save(ctx); err != nil {
		return err
	}

	if err := session.Reload(ctx); err != nil {
		return err
	}

	return nil
}

// expired checks if the session has expired. Must be called with mu held.
func (self *RunningSession) expired() bool {
	if self.session == nil {
		return false
	}
	expiresAt := self.session.ExpiresAt()
	if expiresAt != nil {
		return !time.Now().Before(*expiresAt)
	}
	return false
}

// isConsumed checks if the session resources are consumed. Must be called with mu held.
func (self *RunningSession) isConsumed() bool {
	s := self.session
	if s == nil {
		return false
	}

	t := s.Type()

	// Check expiration date first (applies to all types)
	if self.expired() {
		return true
	}

	// For time-based or time-or-data sessions, check time consumption
	if t == sdkapi.SessionTypeTime || t == sdkapi.SessionTypeTimeOrData {
		if s.RemainingTime() <= 0 {
			return true
		}
	}

	// For data-based or time-or-data sessions, check data consumption
	if t == sdkapi.SessionTypeData || t == sdkapi.SessionTypeTimeOrData {
		if s.DataConsumption() >= s.DataMb() {
			return true
		}
	}

	return false
}
