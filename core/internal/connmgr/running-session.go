package connmgr

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"core/internal/network"
	"core/internal/utils/tc"
	sdkapi "sdk/api"
	jobque "tools/job-que"
)

var (
	sessionQue = jobque.NewJobQue[any]()
)

// SessionEventEmitter interface for emitting session events
type SessionEventEmitter interface {
	emitSessionEvent(event sdkapi.SessionEvent, session sdkapi.IClientSession, device sdkapi.IClientDevice)
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
	log.Printf("[Running Session] UpdateNetworkDetails - DeviceID=%d, OldMAC=%s, NewMAC=%s, OldIP=%s, NewIP=%s",
		self.clntId, self.mac, newMac, self.ip, newIP)

	_, err := sessionQue.Exec(func() (any, error) {
		self.mu.Lock()
		defer self.mu.Unlock()

		oldIP := self.ip
		oldMac := self.mac

		// Check if network details actually changed
		if oldIP == newIP && oldMac == newMac {
			log.Printf("[Running Session] No network changes detected for device %d", self.clntId)
			return nil, nil
		}

		// Update stored values
		self.ip = newIP
		self.mac = newMac

		// Check if LAN changed (IP might be on different network)
		if oldIP != newIP {
			log.Printf("[Running Session] IP changed, checking if LAN changed...")
			newLan, err := network.FindByIp(newIP)
			if err != nil {
				log.Printf("[Running Session] ERROR - Failed to find LAN for new IP %s: %v", newIP, err)
				return nil, err
			}

			// If LAN changed, we need to recreate TC rules on the new interface
			if newLan.Name() != self.lan.Name() {
				log.Printf("[Running Session] LAN changed from %s to %s, recreating TC rules...",
					self.lan.Name(), newLan.Name())

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
					return nil, err
				}
				log.Printf("[Running Session] TC rules recreated successfully")
			} else {
				// Same LAN, just update the filter
				log.Printf("[Running Session] Same LAN, updating TC filter from IP %s to %s", oldIP, newIP)
				if self.tcClassId != nil {
					classid := self.tcClassId.Uint()

					// Remove old filter
					if err := self.lan.DelFilter(oldIP, classid); err != nil {
						log.Printf("[Running Session] WARNING - Failed to delete old filter: %v", err)
					}

					// Create new filter with new IP
					if err := self.lan.CreateFilter(newIP, classid); err != nil {
						log.Printf("[Running Session] ERROR - Failed to create new filter: %v", err)
						return nil, err
					}
					log.Printf("[Running Session] TC filter updated successfully")
				}
			}
		}

		log.Printf("[Running Session] Network details updated successfully - DeviceID=%d, NewMAC=%s, NewIP=%s",
			self.clntId, self.mac, self.ip)
		return nil, nil
	})

	return err
}

func (self *RunningSession) Start(ctx context.Context, s sdkapi.IClientSession) error {
	_, err := sessionQue.Exec(func() (any, error) {
		self.mu.Lock()
		defer self.mu.Unlock()

		// Reload session from database to get latest consumption values
		// This is critical when resuming a paused session to avoid using stale data
		if err := s.Reload(ctx); err != nil {
			return nil, err
		}

		self.session = s

		// Set first start time if this is the first time session is starting
		if s.StartedAt() == nil {
			started := time.Now().UTC()
			s.SetStartedAt(&started)
		}

		// Set resumed time to track current running period
		if s.ResumedAt() == nil {
			resumed := time.Now().UTC()
			s.SetResumedAt(&resumed)

			if err := s.Save(ctx); err != nil {
				return nil, err
			}
		}

		if self.tcClassId == nil {
			if err := self.initTc(); err != nil {
				return nil, err
			}
		} else {
			if err := self.updateTc(); err != nil {
				return nil, err
			}
		}

		if self.timeTimer == nil {
			self.initTimeTimer(s)
			log.Println("Session timer has started...")
		}

		return nil, nil
	})

	return err
}

func (self *RunningSession) Stop(ctx context.Context) error {
	return self.StopWithReason(ctx, false)
}

func (self *RunningSession) StopWithReason(ctx context.Context, expired bool) error {
	_, err := sessionQue.Exec(func() (any, error) {
		self.mu.Lock()

		// Calculate and record elapsed time since resumed_at
		if self.session != nil && self.session.ResumedAt() != nil {
			elapsed := int(time.Since(*self.session.ResumedAt()).Seconds())

			// Add elapsed time to existing consumption
			currentCons := self.session.TimeConsumption()
			self.session.SetTimeCons(currentCons + elapsed)

			log.Printf("Recording elapsed time: %d seconds (total consumption: %d)\n",
				elapsed, currentCons+elapsed)

			// Reset resumed_at to nil since session is stopping
			self.session.SetResumedAt(nil)
		}

		err := self.save(ctx)
		self.cleanUpTimer()

		// Emit session:expired event if session expired (time/data consumed or date passed)
		if expired && self.emitter != nil && self.clnt != nil {
			self.emitter.emitSessionEvent(sdkapi.EventSessionExpired, self.session, self.clnt)
		}

		// Collect callbacks while holding lock
		callbacks := self.callbacks
		self.callbacks = []chan error{}

		// Release lock before sending to channels
		self.mu.Unlock()

		// Send to callbacks without holding lock
		for _, cb := range callbacks {
			cb <- err
		}
		log.Println("Done running callbacks.")

		return nil, err
	})

	return err
}

func (self *RunningSession) CleanupTc() error {
	errCh := make(chan error)

	go func() {
		self.mu.Lock()
		defer self.mu.Unlock()

		if self.tcClassId != nil {
			log.Println("Clean up TC...")
			classid := self.tcClassId.Uint()

			err := self.lan.DelFilter(self.ip, classid)
			if err != nil {
				errCh <- err
				return
			}

			err = self.lan.DelClass(classid)
			self.tcClassId = nil

			errCh <- err
			return
		}

		log.Println("Done cleaning TC.")
		errCh <- nil
	}()

	return <-errCh
}

func (self *RunningSession) UpdateDataConsumption(stats *sdkapi.TrafficData) {
	self.mu.Lock()

	download, dlOK := stats.Download[self.ip]
	upload, upOK := stats.Upload[strings.ToUpper(self.mac)]

	var shouldStop bool
	if dlOK && upOK {
		dataconMb := float64(download.Bytes+upload.Bytes) / (1 * 1000 * 1000)
		log.Println("CONSUMPTION MB: ", dataconMb)
		self.session.IncDataCons(dataconMb)
		self.diffMb += dataconMb

		if self.isConsumed() {
			log.Println("Session data is consumed!!!")
			shouldStop = true
		}
	}

	self.mu.Unlock()

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
				// Periodic save
				self.mu.RLock()
				currentSession := self.session
				currentClnt := self.clnt
				currentEmitter := self.emitter
				self.mu.RUnlock()

				// Calculate elapsed time since resumed_at
				if resumedAt := currentSession.ResumedAt(); resumedAt != nil {
					elapsed := int(time.Since(*resumedAt).Seconds())
					log.Printf("Periodic save: %d seconds elapsed, %d remaining\n",
						elapsed, currentSession.RemainingTime())
				}

				// Save current state (data consumption changes)
				err := self.save(context.Background())
				if err != nil {
					log.Println("Error saving session:", err)
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

	if t == sdkapi.SessionTypeTime || t == sdkapi.SessionTypeTimeOrData {
		isTimeConsumed := s.RemainingTime() <= 0
		return isTimeConsumed || self.expired()
	}

	if t == sdkapi.SessionTypeData || t == sdkapi.SessionTypeTimeOrData {
		isDataConsumed := s.DataConsumption() >= s.DataMb()
		return isDataConsumed || self.expired()
	}

	return false
}
