package sessmgr

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"core/internal/modules/tc"
	"core/internal/network"
	sdkapi "sdk/api"
)

var (
	// tcNftMu serializes all TC and NFT commands globally.
	// TC/NFT subsystem can only handle one command at a time.
	tcNftMu           sync.Mutex
	ErrSessionExpired = errors.New("session expired")
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

func NewRunningSession(clnt sdkapi.IClientDevice, s sdkapi.IClientSession, emitter SessionEventEmitter) (*RunningSession, error) {
	log.Printf("[Running Session] Creating new running session - DeviceID=%d, MAC=%s, IP=%s",
		clnt.ID(), clnt.MacAddr(), clnt.IpAddr())

	lan, err := network.FindByIp(clnt.IpAddr())
	if err != nil {
		log.Printf("[Running Session] ERROR - Failed to find LAN for IP %s: %v", clnt.IpAddr(), err)
		return nil, err
	}

	rs := RunningSession{
		session:   s,
		clnt:      clnt,
		clntId:    clnt.ID(),
		ip:        clnt.IpAddr(),
		mac:       clnt.MacAddr(),
		lan:       lan,
		emitter:   emitter,
		callbacks: []chan error{},
	}

	log.Printf("[Running Session] Running session created successfully - DeviceID=%d, MAC=%s, IP=%s, LAN=%s",
		rs.clntId, rs.mac, rs.ip, lan.Name())

	return &rs, nil
}

type RunningSession struct {
	mu          sync.RWMutex
	clnt        sdkapi.IClientDevice
	clntId      int64
	ip          string
	mac         string
	lan         *network.NetworkLan
	tcClassId   *tc.TcClassId
	tcFilter    *tc.TcFilter
	timeTimer   *time.Timer
	timerCancel context.CancelFunc
	timerCtx    context.Context
	session     sdkapi.IClientSession
	emitter     SessionEventEmitter
	diffMb      float64
	callbacks   []chan error
}

func (self *RunningSession) ClientId() int64 {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.clntId
}

func (self *RunningSession) GetSession() sdkapi.IClientSession {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.session
}

func (self *RunningSession) Lan() *network.NetworkLan {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.lan
}

func (self *RunningSession) Done() <-chan error {
	self.mu.Lock()
	defer self.mu.Unlock()

	ch := make(chan error)
	self.callbacks = append(self.callbacks, ch)
	return ch
}

func (self *RunningSession) DiffMb() (mb float64) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.diffMb
}

// UpdateNetworkDetails updates the MAC and IP address when device network details change
func (self *RunningSession) UpdateNetworkDetails(ctx context.Context, newMac, newIP string) error {
	self.mu.RLock()
	contextInfo := fmt.Sprintf("DeviceID=%d, OldMAC=%s, NewMAC=%s, OldIP=%s, NewIP=%s",
		self.clntId, self.mac, newMac, self.ip, newIP)
	oldIP := self.ip
	oldMac := self.mac
	self.mu.RUnlock()

	log.Printf("[Running Session] UpdateNetworkDetails - %s", contextInfo)

	// Check if network details actually changed
	if oldIP == newIP && oldMac == newMac {
		log.Printf("[Running Session] No network changes detected for device %d", self.clntId)
		return nil
	}

	// Update stored values (use self.mu)
	self.mu.Lock()
	self.ip = newIP
	self.mac = newMac
	self.mu.Unlock()

	// Check if LAN changed (IP might be on different network)
	if oldIP != newIP {
		log.Printf("[Running Session] IP changed, checking if LAN changed...")
		newLan, err := network.FindByIp(newIP)
		if err != nil {
			log.Printf("[Running Session] ERROR - Failed to find LAN for new IP %s: %v", newIP, err)
			return err
		}

		self.mu.RLock()
		currentLanName := self.lan.Name()
		self.mu.RUnlock()

		// If LAN changed, we need to recreate TC rules on the new interface
		if newLan.Name() != currentLanName {
			log.Printf("[Running Session] LAN changed from %s to %s, recreating TC rules...",
				currentLanName, newLan.Name())

			// TC operations - use global tcNftMu lock
			err := withTcNftLock("TC Network Update (LAN changed)", contextInfo, func() error {
				self.mu.Lock()
				defer self.mu.Unlock()

				// Clean up old TC rules
				if self.tcClassId != nil {
					classid := self.tcClassId.Uint()
					log.Printf("[Running Session] Removing old TC filter for IP %s", oldIP)
					if err := self.lan.DelFilter(oldIP, classid); err != nil {
						log.Printf("[Running Session] WARNING - Failed to delete old filter: %v", err)
					}

					log.Printf("[Running Session] Removing old TC class %d", classid)
					if err := self.lan.DelClass(classid); err != nil {
						log.Printf("[Running Session] WARNING - Failed to delete old class: %v", err)
					}
					self.tcClassId = nil
				}

				// Update LAN reference
				self.lan = newLan

				// Recreate TC rules on new interface
				log.Printf("[Running Session] Creating new TC rules on interface %s", newLan.Name())
				if err := self.initTc(); err != nil {
					log.Printf("[Running Session] ERROR - Failed to create TC rules: %v", err)
					return err
				}
				log.Printf("[Running Session] TC rules recreated successfully")
				return nil
			})
			if err != nil {
				return err
			}
		} else {
			// Same LAN, just update the filter
			log.Printf("[Running Session] Same LAN, updating TC filter from IP %s to %s", oldIP, newIP)

			err := withTcNftLock("TC Network Update (same LAN)", contextInfo, func() error {
				self.mu.Lock()
				defer self.mu.Unlock()

				if self.tcClassId != nil {
					classid := self.tcClassId.Uint()

					// Remove old filter
					if err := self.lan.DelFilter(oldIP, classid); err != nil {
						log.Printf("[Running Session] WARNING - Failed to delete old filter: %v", err)
					}

					// Create new filter with new IP
					if err := self.lan.CreateFilter(newIP, classid); err != nil {
						log.Printf("[Running Session] ERROR - Failed to create new filter: %v", err)
						return err
					}
					log.Printf("[Running Session] TC filter updated successfully")
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	log.Printf("[Running Session] Network details updated successfully - DeviceID=%d, NewMAC=%s, NewIP=%s",
		self.clntId, newMac, newIP)
	return nil
}

func (self *RunningSession) Start(ctx context.Context, s sdkapi.IClientSession) error {
	contextInfo := fmt.Sprintf("DeviceID=%d, SessionID=%d, MAC=%s, IP=%s",
		self.clntId, s.ID(), self.mac, self.ip)

	log.Printf("[Running Session] Start - %s", contextInfo)

	// 1. DB operations - no TC/NFT lock needed
	dbStart := time.Now()
	if err := s.Reload(ctx); err != nil {
		return fmt.Errorf("failed to reload session: %w", err)
	}
	logSlowOperation("DB Reload", dbStart, 2*time.Second, contextInfo)

	// 2. Update session state (use self.mu for in-memory state)
	self.mu.Lock()
	self.session = s

	// Set first start time if this is the first time session is starting
	timeNow := time.Now().UTC()
	if s.StartedAt() == nil {
		s.SetStartedAt(&timeNow)
		s.SetResumedAt(&timeNow)
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
	self.mu.RLock()
	hasTcClass := self.tcClassId != nil
	self.mu.RUnlock()

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
	self.mu.RLock()
	contextInfo := fmt.Sprintf("DeviceID=%d, SessionID=%d, Expired=%v",
		self.clntId, self.session.ID(), expired)
	self.mu.RUnlock()

	log.Printf("[Running Session] StopWithReason - %s", contextInfo)

	// 1. Calculate and record elapsed time (use session's own mutex)
	self.mu.RLock()
	session := self.session
	self.mu.RUnlock()

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

	// 2. DB save - no TC/NFT lock needed
	dbStart := time.Now()
	saveErr := self.save(ctx)
	logSlowOperation("DB Save (Stop)", dbStart, 2*time.Second, contextInfo)

	// 3. Clean up timer (use self.mu)
	self.mu.Lock()
	self.cleanUpTimer()

	// Collect callbacks while holding lock
	callbacks := self.callbacks
	self.callbacks = []chan error{}

	// Get references for event emission
	emitter := self.emitter
	clnt := self.clnt
	self.mu.Unlock()

	// 4. Emit events - no lock needed
	if expired && emitter != nil && clnt != nil {
		emitter.emitSessionEvent(sdkapi.EventSessionBeforeExpired, session, clnt)
		emitter.emitSessionEvent(sdkapi.EventSessionExpired, session, clnt)
	}

	// 5. Determine the error to return to callbacks
	var callbackErr error
	if expired {
		callbackErr = ErrSessionExpired
	} else {
		callbackErr = saveErr
	}

	// 6. Send to callbacks - no lock needed
	for _, cb := range callbacks {
		cb <- callbackErr
	}
	log.Println("Done running callbacks.")

	log.Printf("[Running Session] StopWithReason completed - %s", contextInfo)
	return callbackErr
}

func (self *RunningSession) CleanupTc() error {
	self.mu.RLock()
	contextInfo := fmt.Sprintf("DeviceID=%d, IP=%s", self.clntId, self.ip)
	self.mu.RUnlock()

	errCh := make(chan error)

	go func() {
		err := withTcNftLock("CleanupTc", contextInfo, func() error {
			self.mu.Lock()
			defer self.mu.Unlock()

			if self.tcClassId != nil {
				log.Println("Clean up TC...")
				classid := self.tcClassId.Uint()

				if err := self.lan.DelFilter(self.ip, classid); err != nil {
					return err
				}

				if err := self.lan.DelClass(classid); err != nil {
					self.tcClassId = nil
					return err
				}
				self.tcClassId = nil
			}

			log.Println("Done cleaning TC.")
			return nil
		})
		errCh <- err
	}()

	return <-errCh
}

func (self *RunningSession) UpdateDataConsumption(stats *sdkapi.TrafficData) {
	// Read IP and MAC with RLock (read-only access)
	self.mu.RLock()
	ip := self.ip
	mac := self.mac
	session := self.session
	self.mu.RUnlock()

	// Look up traffic stats (no lock needed)
	download, dlOK := stats.Download[ip]
	upload, upOK := stats.Upload[strings.ToUpper(mac)]

	var shouldStop bool
	if dlOK && upOK {
		dataconMb := float64(download.Bytes+upload.Bytes) / (1 * 1000 * 1000)
		log.Println("CONSUMPTION MB: ", dataconMb)

		// IncDataCons uses session's internal RWMutex
		session.IncDataCons(dataconMb)

		// Update diffMb with write lock
		self.mu.Lock()
		self.diffMb += dataconMb
		self.mu.Unlock()

		// Check if consumed (uses RLock internally)
		self.mu.RLock()
		if self.isConsumed() {
			log.Println("Session data is consumed!!!")
			shouldStop = true
		}
		self.mu.RUnlock()
	}

	// Call StopWithReason() after releasing the lock to avoid deadlock
	if shouldStop {
		go self.StopWithReason(context.Background(), true)
	}
}

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
		// Periodic save ticker (every 15 seconds)
		saveTicker := time.NewTicker(15 * time.Second)
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
				// Periodic save - no queue, direct call
				self.mu.RLock()
				currentSession := self.session
				currentClnt := self.clnt
				currentEmitter := self.emitter
				deviceID := self.clntId
				self.mu.RUnlock()

				// Calculate elapsed time since resumed_at
				var elapsed int
				if resumedAt := currentSession.ResumedAt(); resumedAt != nil {
					elapsed = int(time.Since(*resumedAt).Seconds())
					log.Printf("Periodic save: %d seconds elapsed, %d remaining\n",
						elapsed, currentSession.RemainingTime())
				}

				// Emit before updated hook
				if currentEmitter != nil && currentClnt != nil {
					if err := currentEmitter.emitSessionEvent(sdkapi.EventSessionBeforeUpdated, currentSession, currentClnt); err != nil {
						log.Println("Before update hook failed:", err)
						go self.Stop(context.Background())
						return
					}
				}

				// Direct save - no lock needed (DB handles concurrency)
				contextInfo := fmt.Sprintf("DeviceID=%d, SessionID=%d, Elapsed=%ds",
					deviceID, currentSession.ID(), elapsed)

				dbStart := time.Now()
				saveErr := self.save(context.Background())
				logSlowOperation("Periodic Save", dbStart, 2*time.Second, contextInfo)

				if saveErr != nil {
					log.Printf("[ERROR] Periodic save failed: %v - STOPPING SESSION", saveErr)
					go self.Stop(context.Background())
					return
				}

				// Emit session:updated event after successful save
				if currentEmitter != nil && currentClnt != nil {
					currentEmitter.emitSessionEvent(sdkapi.EventSessionUpdated, currentSession, currentClnt)
				}

				// Reset diff counter for data
				self.mu.Lock()
				self.diffMb = 0
				self.mu.Unlock()
			}
		}
	}()
}

func (self *RunningSession) initTc() error {
	classid := tc.GetAvailableId()
	defer classid.Cancel()

	lan := self.lan
	s := self.session

	err := lan.CreateClass(classid.Uint(), s.DownMbits(), s.UpMbits())
	if err != nil {
		return err
	}

	err = lan.CreateFilter(self.ip, classid.Uint())
	if err != nil {
		lan.DelClass(classid.Uint())
		return err
	}

	classid.Commit()
	self.tcClassId = &classid

	return nil
}

func (self *RunningSession) updateTc() error {
	var (
		downMbits = self.session.DownMbits()
		upMbits   = self.session.UpMbits()
		useGlobal = self.session.UseGlobalSpeed()
	)

	if useGlobal {
		lan, err := network.FindByIp(self.ip)
		if err != nil {
			return err
		}

		d, u := lan.Bandwidth()
		downMbits, upMbits = int(d), int(u)
	}

	return self.lan.ChangeClass(self.tcClassId.Uint(), downMbits, upMbits)
}

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

func (self *RunningSession) save(ctx context.Context) error {
	if self.session != nil {
		if err := self.session.Save(ctx); err != nil {
			return err
		}

		if err := self.session.Reload(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (self *RunningSession) expired() (ok bool) {
	expiresAt := self.session.ExpiresAt()
	if expiresAt != nil {
		return !time.Now().Before(*expiresAt)
	}
	return false
}

func (self *RunningSession) isConsumed() bool {
	s := self.session
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
