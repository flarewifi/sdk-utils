package sessmgr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"core/db"
	"core/db/models"
	"core/internal/modules/nftables"
	"core/internal/network"
	sdkapi "sdk/api"
)

var (
// Removed hardcoded error messages - now translated at runtime
)

func NewSessionsMgr(dtb *db.Database, mdl *models.Models) *SessionsMgr {
	sessionMgr := &SessionsMgr{
		db:       dtb,
		mdl:      mdl,
		sessions: sync.Map{},
	}
	return sessionMgr
}

type SessionsMgr struct {
	coreAPI    sdkapi.IPluginApi
	pluginsMgr sdkapi.IPluginsMgrApi
	db         *db.Database
	mdl        *models.Models
	sessions   sync.Map

	// Event callbacks - protected by dedicated mutex to prevent race on Load-Modify-Store
	eventMu               sync.Mutex
	sessionEventCallbacks map[sdkapi.SessionEvent][]func(data sdkapi.SessionEventData) error
	clientEventCallbacks  map[sdkapi.ClientEvent][]func(clnt sdkapi.IClientDevice) error
}

func (self *SessionsMgr) SetCoreAPI(api sdkapi.IPluginApi) {
	self.coreAPI = api
	if api != nil {
		self.pluginsMgr = api.PluginsMgr()
	}
}

func (self *SessionsMgr) Init(ctx context.Context) error {
	// First, update consumption for all running sessions
	err := self.db.Queries.BulkUpdateTimeConsumption(ctx)
	if err != nil {
		return fmt.Errorf("failed to update consumption before reset: %w", err)
	}

	// Then reset all resumed_at fields to NULL
	err = self.db.Queries.ResetAllResumedAt(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset resumed_at fields: %w", err)
	}

	// Reset all device connection statuses to disconnected
	err = self.db.Queries.ResetAllDeviceStatuses(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset device statuses: %w", err)
	}

	return nil
}

func (self *SessionsMgr) OnSessionEvent(event sdkapi.SessionEvent, callback func(data sdkapi.SessionEventData) error) {
	self.eventMu.Lock()
	defer self.eventMu.Unlock()

	if self.sessionEventCallbacks == nil {
		self.sessionEventCallbacks = make(map[sdkapi.SessionEvent][]func(data sdkapi.SessionEventData) error)
	}
	self.sessionEventCallbacks[event] = append(self.sessionEventCallbacks[event], callback)
}

func (self *SessionsMgr) OnClientEvent(event sdkapi.ClientEvent, callback func(clnt sdkapi.IClientDevice) error) {
	self.eventMu.Lock()
	defer self.eventMu.Unlock()

	if self.clientEventCallbacks == nil {
		self.clientEventCallbacks = make(map[sdkapi.ClientEvent][]func(clnt sdkapi.IClientDevice) error)
	}
	self.clientEventCallbacks[event] = append(self.clientEventCallbacks[event], callback)
}

func (self *SessionsMgr) emitSessionEvent(event sdkapi.SessionEvent, session sdkapi.IClientSession, device sdkapi.IClientDevice) error {
	// Take a snapshot of callbacks under lock to avoid holding lock during callback execution
	self.eventMu.Lock()
	callbacks := self.sessionEventCallbacks[event]
	// Copy slice header so we don't hold the lock during callbacks
	callbacksCopy := make([]func(data sdkapi.SessionEventData) error, len(callbacks))
	copy(callbacksCopy, callbacks)
	self.eventMu.Unlock()

	data := sdkapi.SessionEventData{
		Session: session,
		Device:  device,
	}
	for _, callback := range callbacksCopy {
		if err := callback(data); err != nil {
			return err
		}
	}
	return nil
}

// EmitSessionEvent is a public wrapper for emitSessionEvent to allow API layer to trigger events
func (self *SessionsMgr) EmitSessionEvent(event sdkapi.SessionEvent, session sdkapi.IClientSession, device sdkapi.IClientDevice) error {
	return self.emitSessionEvent(event, session, device)
}

func (self *SessionsMgr) emitClientEvent(event sdkapi.ClientEvent, clnt sdkapi.IClientDevice) error {
	// Take a snapshot of callbacks under lock to avoid holding lock during callback execution
	self.eventMu.Lock()
	callbacks := self.clientEventCallbacks[event]
	callbacksCopy := make([]func(clnt sdkapi.IClientDevice) error, len(callbacks))
	copy(callbacksCopy, callbacks)
	self.eventMu.Unlock()

	for _, callback := range callbacksCopy {
		if err := callback(clnt); err != nil {
			return err
		}
	}
	return nil
}

func (self *SessionsMgr) ListenTraffic(trfk *network.TrafficMgr) {
	// Use a single goroutine to process traffic updates sequentially.
	// This prevents unbounded goroutine spawning under high traffic.
	// Each update iterates all running sessions, which is fast (in-memory).
	go func() {
		for data := range trfk.Listen() {
			dataCopy := data // Copy loop variable for safety
			self.sessions.Range(func(key, value any) bool {
				rs := value.(*RunningSession)
				rs.UpdateDataConsumption(&dataCopy)
				return true
			})
		}
	}()
}

func (self *SessionsMgr) ReloadSessions(ctx context.Context, iface string) error {
	errCh := make(chan error, 1) // Buffered to prevent goroutine leak

	go func() {
		var rangeErr error
		self.sessions.Range(func(key, value any) bool {
			rs := value.(*RunningSession)
			lan := rs.Lan()

			if lan.Name() == iface {
				// Skip sessions that are stopped or in the process of stopping.
				// ReloadSessions is called when network interfaces change, but a session
				// may be concurrently stopping (e.g., timer expired). Calling Start() on
				// a stopping session would clear the stopped flag and create inconsistent state.
				if rs.IsStopped() {
					log.Printf("[SessionsMgr] ReloadSessions: skipping stopped session for device %d", rs.ClientId())
					return true
				}

				cs := rs.GetSession()
				err := cs.Reload(ctx)
				if err != nil {
					rangeErr = err
					return false
				}

				err = rs.Start(ctx, cs)
				if err != nil {
					rangeErr = err
					return false
				}
			}

			return true
		})

		errCh <- rangeErr // Always sends exactly once (nil or error)
	}()

	return <-errCh
}

func (self *SessionsMgr) StopSessions(ctx context.Context, iface string, reason string) {
	self.sessions.Range(func(key, value any) bool {
		rs := value.(*RunningSession)

		// Read network state atomically from the RunningSession.
		// This is the authoritative state (updated inside tcNftMu+mu by UpdateNetworkDetails).
		lan := rs.Lan()
		if lan == nil {
			return true
		}

		if lan.Name() == iface {
			// Re-read IP/MAC at point of use to minimize staleness window.
			// These are atomic reads from the RunningSession's network state.
			ip := rs.IpAddr()
			mac := rs.MacAddr()

			err := nftables.Disconnect(ip, mac)
			if err != nil {
				log.Printf("[SessionsMgr] StopSessions: failed to disconnect device IP=%s MAC=%s: %v", ip, mac, err)
			}

			if err := rs.Stop(ctx); err != nil {
				log.Printf("[SessionsMgr] StopSessions: failed to stop session for device IP=%s: %v", ip, err)
			}

			// Clean up TC classes/filters and restore class ID to pool
			if err := rs.CleanupTc(); err != nil {
				log.Printf("[SessionsMgr] StopSessions: failed to cleanup TC for device IP=%s: %v", ip, err)
			}

			// Remove from sessions map
			self.sessions.Delete(key)
		}

		return true
	})
}

func (self *SessionsMgr) Connect(ctx context.Context, clnt sdkapi.IClientDevice, notify string) error {
	if clnt.Status() == sdkapi.DeviceStatusBlocked {
		return errors.New(self.coreAPI.Translate("error", "Device is blocked"))
	}

	// Launch session loop - handles nftables, session start, events, and chaining
	resultCh := make(chan error, 1)
	go self.loopSessions(resultCh, clnt, notify)

	// Wait for result with context cancellation support
	var err error
	select {
	case err = <-resultCh:
		// Normal case - received result from loopSessions
	case <-ctx.Done():
		// Context cancelled (e.g., HTTP timeout) - loopSessions continues in background
		return ctx.Err()
	}

	if err == nil {
		err = clnt.Update(ctx, sdkapi.UpdateDeviceParams{
			UUID:     clnt.UUID(),
			Mac:      clnt.MacAddr(),
			Ip:       clnt.IpAddr(),
			Hostname: clnt.Hostname(),
			Status:   sdkapi.DeviceStatusConnected,
		})
		if err != nil {
			err = fmt.Errorf("unable to update device status to connected: %w", err)
		}
	}
	return err
}

func (self *SessionsMgr) Disconnect(ctx context.Context, clnt sdkapi.IClientDevice, notify string) error {
	// Session events (EventSessionDisconnected) are emitted by StopWithReason() inside endSession().
	err := self.endSession(ctx, clnt)
	if err != nil {
		return err
	}

	clnt.Emit(string(sdkapi.EventSessionDisconnected), []byte(notify))
	self.emitClientEvent(sdkapi.EventClientDisconnected, clnt)

	return clnt.Update(ctx, sdkapi.UpdateDeviceParams{
		UUID:     clnt.UUID(),
		Mac:      clnt.MacAddr(),
		Ip:       clnt.IpAddr(),
		Hostname: clnt.Hostname(),
		Status:   sdkapi.DeviceStatusDisconnected,
	})
}

func (self *SessionsMgr) IsConnected(clnt sdkapi.IClientDevice) (connected bool) {
	return nftables.IsConnected(clnt.MacAddr())
}

func (self *SessionsMgr) CurrSession(clnt sdkapi.IClientDevice) (cs sdkapi.IClientSession, ok bool) {
	v, ok := self.sessions.Load(clnt.ID())
	if !ok {
		return nil, false
	}

	rs, ok := v.(*RunningSession)
	if !ok {
		return nil, false
	}

	return rs.GetSession(), true
}

func (self *SessionsMgr) loopSessions(resultCh chan<- error, clnt sdkapi.IClientDevice, notify string) {
	var callbackDone atomic.Bool
	ctx := context.Background()

	// Loop condition: continue while connected OR until first session starts
	// This allows the first iteration to run before nftables rules are added
	for nftables.IsConnected(clnt.MacAddr()) || !callbackDone.Load() {
		// Get next available session
		cs, err := self.GetSession(ctx, clnt)
		if err != nil {
			if !callbackDone.Load() {
				// First attempt failed - user sees error immediately
				resultCh <- err
				callbackDone.Store(true)
			} else {
				// Session chaining failed - no more sessions available
				self.Disconnect(ctx, clnt, self.coreAPI.Translate("info", "No more sessions available"))
			}
			return
		}

		// Get or create running session
		rs, ok := self.getRunningSession(clnt)
		if !ok {
			rs, err = NewRunningSession(clnt, cs, self)
			if err != nil {
				if !callbackDone.Load() {
					resultCh <- err
					callbackDone.Store(true)
				} else {
					self.Disconnect(ctx, clnt, err.Error())
				}
				return
			}
			self.sessions.Store(clnt.ID(), rs)
		}

		// Start the session (this also sets up TC classes/filters)
		err = rs.Start(ctx, cs)
		if err != nil {
			if !callbackDone.Load() {
				// First session start failed - user sees error immediately
				resultCh <- err
				callbackDone.Store(true)
			} else {
				// Chained session start failed - disconnect
				self.Disconnect(ctx, clnt, err.Error())
			}
			return
		}

		// First successful start - add firewall rules and emit events
		if !callbackDone.Load() {
			// Add firewall rules to allow internet access
			if err := nftables.Connect(clnt.IpAddr(), clnt.MacAddr()); err != nil {
				// nftables failed - stop the session, cleanup TC, and return error
				rs.Stop(ctx)
				rs.CleanupTc()
				self.sessions.Delete(clnt.ID())
				resultCh <- err
				callbackDone.Store(true)
				return
			}

			// Emit connection events
			clnt.Emit(string(sdkapi.EventSessionConnected), []byte(notify))
			if session, ok := self.CurrSession(clnt); ok {
				self.emitSessionEvent(sdkapi.EventSessionConnected, session, clnt)
			}
			self.emitClientEvent(sdkapi.EventClientConnected, clnt)

			// Signal success to Connect()
			resultCh <- nil
			callbackDone.Store(true)
		}

		// Wait for session to end
		err = <-rs.Done()

		// Handle session end
		if err != nil {
			if errors.Is(err, ErrSessionExpired) || errors.Is(err, ErrSessionStopped) {
				// Session expired or was stopped (e.g., via UpdateTime(0)) - reset state and continue to try next session
				// TC class/filter are preserved for reuse with the next session
				log.Printf("Session ended for device %s (reason: %v), checking for next available session...", clnt.MacAddr(), err)
				rs.Reset()

				// Check if this loop has been superseded by a new Connect() call.
				// UpdateDevice calls Disconnect (which deletes from sessions map and stops rs),
				// then Connect (which spawns a NEW loopSessions with a new RunningSession).
				// If the RunningSession in the map is no longer ours, we've been replaced — exit.
				currentRs, stillInMap := self.getRunningSession(clnt)
				if !stillInMap || currentRs != rs {
					log.Printf("[loopSessions] Session loop superseded for device %s (replaced by new Connect), exiting", clnt.MacAddr())
					return
				}

				continue
			}
			// Other error - disconnect
			self.Disconnect(ctx, clnt, err.Error())
			return
		}

		// Session ended without error - continue loop to check for next session
	}

	// Loop exited because nftables.IsConnected returned false
	// Device was disconnected externally
}

func (self *SessionsMgr) getRunningSession(clnt sdkapi.IClientDevice) (rs *RunningSession, ok bool) {
	v, ok := self.sessions.Load(clnt.ID())
	if !ok {
		return nil, false
	}

	rs, ok = v.(*RunningSession)
	if !ok {
		return nil, false
	}

	return rs, true
}

// GetRunningSession returns the running session for a client device (public wrapper)
func (self *SessionsMgr) GetRunningSession(clnt sdkapi.IClientDevice) (rs *RunningSession, ok bool) {
	return self.getRunningSession(clnt)
}

func (self *SessionsMgr) endSession(ctx context.Context, clnt sdkapi.IClientDevice) error {
	if nftables.IsConnected(clnt.MacAddr()) {
		if err := nftables.Disconnect(clnt.IpAddr(), clnt.MacAddr()); err != nil {
			return err
		}
	}

	rs, ok := self.getRunningSession(clnt)
	if ok {
		if err := rs.Stop(ctx); err != nil {
			return err
		}

		if err := rs.CleanupTc(); err != nil {
			return err
		}
	}

	self.sessions.Delete(clnt.ID())
	return nil
}

func (self *SessionsMgr) GetSession(ctx context.Context, clnt sdkapi.IClientDevice) (sdkapi.IClientSession, error) {
	localClient := clnt.(*ClientDevice)
	s, err := self.mdl.Session().AvailableForDevice(ctx, localClient.id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New(self.coreAPI.Translate("error", "No more available sessions"))
		}
		return nil, err
	}

	return self.wrapModelSession(s), nil
}

// SessionSummary returns the total remaining time/data from ALL sessions for a client device.
// The database queries return the total based on saved consumption values.
// We need to subtract both elapsed time and unsaved data consumption for running sessions.
func (self *SessionsMgr) SessionSummary(ctx context.Context, clnt sdkapi.IClientDevice) (*sdkapi.ClientSessionSummary, error) {
	summary, err := self.mdl.Session().Summary(ctx, clnt.ID())
	if err != nil {
		return nil, err
	}

	// Check if there's a running session
	rs, ok := self.getRunningSession(clnt)
	if !ok {
		// No running session, return database totals as-is
		return summary, nil
	}

	// Calculate elapsed time for the running session since resumed_at.
	// Use a single GetSession() call and a single ResumedAt() snapshot to avoid
	// a race where SnapshotTimeCons sets resumedAt to nil between the nil check
	// and the dereference, which would cause a nil pointer panic.
	var elapsedSecs int = 0
	session := rs.GetSession()
	resumedAt := session.ResumedAt()
	if resumedAt != nil {
		elapsedSecs = int(time.Since(*resumedAt).Seconds())
	}

	// Get unsaved data consumption diff (data consumed but not yet written to DB)
	mbDiff := rs.DiffMb()

	// Subtract both elapsed time and unsaved data consumption
	remainingTime := summary.RemainingTimeSecs - elapsedSecs
	remainingData := summary.RemainingDataMbytes - mbDiff

	// Ensure we don't go below zero
	remainingTime = max(remainingTime, 0)
	remainingData = max(remainingData, 0)

	return &sdkapi.ClientSessionSummary{
		RemainingTimeSecs:   remainingTime,
		RemainingDataMbytes: remainingData,
	}, nil
}

// FindDeviceByID finds a device by its database ID and wraps it into an IClientDevice object.
func (self *SessionsMgr) FindDeviceByID(ctx context.Context, deviceID int64) (sdkapi.IClientDevice, error) {
	d, err := self.mdl.Device().Find(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	return self.wrapModelDevice(d), nil
}

// FindDeviceByUUID finds a device by its UUID and wraps it into an IClientDevice object.
func (self *SessionsMgr) FindDeviceByUUID(ctx context.Context, uuid string) (sdkapi.IClientDevice, error) {
	d, err := self.mdl.Device().FindByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	return self.wrapModelDevice(d), nil
}

// FindSessionByID finds a session by its database ID and wraps it into an IClientSession object.
func (self *SessionsMgr) FindSessionByID(ctx context.Context, sessionID int64) (sdkapi.IClientSession, error) {
	s, err := self.mdl.Session().Find(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return self.wrapModelSession(s), nil
}

// FindSessionByUUID finds a session by its UUID and wraps it into an IClientSession object.
func (self *SessionsMgr) FindSessionByUUID(ctx context.Context, uuid string) (sdkapi.IClientSession, error) {
	s, err := self.mdl.Session().FindByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	return self.wrapModelSession(s), nil
}

// NewClientSession wraps session data into an IClientSession object without performing
// additional database queries.
func (self *SessionsMgr) NewClientSession(params sdkapi.NewClientSessionParams) sdkapi.IClientSession {
	// Create a models.Session from the params using BuildSession
	s := models.BuildSession(models.BuildSessionParams{
		DB:          self.db,
		Models:      self.mdl,
		ID:          params.ID,
		UUID:        params.UUID,
		ProviderPkg: params.ProviderPkg,
		DeviceID:    params.DeviceID,
		SessionType: string(params.SessionType),
		TimeSecs:    params.TimeSecs,
		DataMbytes:  params.DataMbytes,
		TimeCons:    params.ConsumptionSecs,
		DataCons:    params.ConsumptionMb,
		StartedAt:   params.StartedAt,
		ResumedAt:   params.ResumedAt,
		ExpDays:     params.ExpDays,
		DownMbits:   params.DownMbits,
		UpMbits:     params.UpMbits,
		UseGlobal:   params.UseGlobal,
		CreatedAt:   params.CreatedAt,
		UpdatedAt:   params.UpdatedAt,
	})
	return self.wrapModelSession(s)
}

// wrapModelSession wraps a models.Session into an IClientSession with save callback.
// This is the internal helper used by all session-wrapping methods.
func (self *SessionsMgr) wrapModelSession(s *models.Session) *ClientSession {
	cs := NewClientSession(self.db, self.mdl, self.coreAPI.PluginsMgr(), s)
	cs.SetOnSave(self.createSessionSaveCallback())
	return cs
}

// wrapModelDevice wraps a models.Device into an IClientDevice.
// This is the internal helper used by all device-wrapping methods.
func (self *SessionsMgr) wrapModelDevice(d *models.Device) *ClientDevice {
	clnt := NewClientDevice(self.db, self.mdl, d)
	clnt.SetIsConnectedFunc(self.isDeviceConnected)
	return clnt
}

// isDeviceConnected checks if a device has a running session (resumed_at IS NOT NULL).
func (self *SessionsMgr) isDeviceConnected(deviceID int64) bool {
	// Check in-memory running sessions first (faster)
	if _, ok := self.sessions.Load(deviceID); ok {
		return true
	}
	return false
}

// NewClientDevice wraps device data into an IClientDevice object without performing
// additional database queries.
func (self *SessionsMgr) NewClientDevice(params sdkapi.NewDeviceParams) sdkapi.IClientDevice {
	// Create a models.Device from the params using BuildDevice
	d := models.BuildDevice(models.BuildDeviceParams{
		DB:        self.db,
		Models:    self.mdl,
		ID:        params.ID,
		UUID:      params.UUID,
		MacAddr:   params.MacAddress,
		IpAddr:    params.IpAddress,
		Hostname:  params.Hostname,
		Status:    params.Status,
		CreatedAt: params.CreatedAt,
		UpdatedAt: params.UpdatedAt,
	})
	return self.wrapModelDevice(d)
}

// ============================================================================
// SESSION SAVE CALLBACK - Handles side effects when session.Save() is called
// ============================================================================

// createSessionSaveCallback creates a callback for ClientSession.Save() to notify about changes.
// This callback applies side effects to running sessions and emits events.
func (self *SessionsMgr) createSessionSaveCallback() sdkapi.SessionSaveCallback {
	return func(params sdkapi.SessionSaveParams) error {
		return self.handleSessionSaved(params)
	}
}

// handleSessionSaved applies side effects after a session is saved.
// For running sessions: resets timer (if time changed), updates TC rules (if bandwidth changed).
// For all sessions: emits EventSessionUpdated.
func (self *SessionsMgr) handleSessionSaved(params sdkapi.SessionSaveParams) error {
	session := params.Session
	changed := params.ChangedFields

	// Get all session data in a single atomic snapshot to avoid multiple getter calls
	sessionData := session.Data()

	// Check if this is a running session
	rs, clnt, isRunning := self.getRunningSessionBySessionID(sessionData.ID)

	if isRunning {
		// Apply side effects to running session
		// Time changed: timeSecs or timeCons
		if changed.TimeSecs || changed.TimeCons {
			if err := rs.ApplyTimeUpdate(ApplyTimeUpdateParams{
				Ctx:           params.Ctx,
				RemainingSecs: sessionData.RemainingTime,
			}); err != nil {
				return err
			}
		}
		// Data changed: dataMb or dataCons
		if changed.DataMb || changed.DataCons {
			// Check if session is now consumed after data update
			if err := rs.ApplyDataUpdate(params.Ctx); err != nil {
				return err
			}
		}
		// Bandwidth changed: downMbits, upMbits, or useGlobalSpeed
		if changed.DownMbits || changed.UpMbits || changed.UseGlobalSpeed {
			if err := rs.ApplyBandwidthUpdate(ApplyBandwidthUpdateParams{
				Ctx:       params.Ctx,
				DownMbits: sessionData.DownMbits,
				UpMbits:   sessionData.UpMbits,
				UseGlobal: sessionData.UseGlobalSpeed,
			}); err != nil {
				return err
			}
		}
	} else {
		// Not running - find device for event emission
		device, err := self.mdl.Device().Find(params.Ctx, sessionData.DeviceID)
		if err != nil {
			return fmt.Errorf("failed to find device for session: %w", err)
		}
		clnt = self.wrapModelDevice(device)
	}

	// Emit event
	self.emitSessionEvent(sdkapi.EventSessionChanged, session, clnt)
	return nil
}

// ============================================================================
// SESSION UPDATE METHODS
// ============================================================================

// getRunningSessionBySessionID finds a running session by its session ID.
// Returns the running session, the client device, and whether it was found.
func (self *SessionsMgr) getRunningSessionBySessionID(sessionID int64) (*RunningSession, sdkapi.IClientDevice, bool) {
	var foundRs *RunningSession
	var foundClnt sdkapi.IClientDevice

	self.sessions.Range(func(key, value any) bool {
		rs := value.(*RunningSession)
		session := rs.GetSession()
		if session != nil && session.ID() == sessionID {
			foundRs = rs
			foundClnt = rs.clnt
			return false // Stop iteration
		}
		return true // Continue iteration
	})

	if foundRs != nil {
		return foundRs, foundClnt, true
	}
	return nil, nil, false
}

// ListRunningSessions returns all currently active (running) sessions.
// These are sessions that are actively connected and consuming time/data.
func (self *SessionsMgr) ListRunningSessions() ([]sdkapi.IClientSession, error) {
	var sessions []sdkapi.IClientSession

	self.sessions.Range(func(key, value any) bool {
		rs := value.(*RunningSession)
		// Skip sessions that are stopped or in the process of stopping
		if rs.IsStopped() {
			return true // Continue iteration
		}
		session := rs.GetSession()
		if session != nil {
			sessions = append(sessions, session)
		}
		return true // Continue iteration
	})

	return sessions, nil
}

// FindRunningSessionByUUID finds a currently running session by its UUID.
// Returns the session and true if found, or nil and false if no running session
// exists with the given UUID.
func (self *SessionsMgr) FindRunningSessionByUUID(uuid string) (sdkapi.IClientSession, bool) {
	var foundSession sdkapi.IClientSession

	self.sessions.Range(func(key, value any) bool {
		rs := value.(*RunningSession)
		// Skip sessions that are stopped or in the process of stopping
		if rs.IsStopped() {
			return true // Continue iteration
		}
		session := rs.GetSession()
		if session != nil && session.UUID() == uuid {
			foundSession = session
			return false // Stop iteration
		}
		return true // Continue iteration
	})

	if foundSession != nil {
		return foundSession, true
	}
	return nil, false
}

// UpdateInterfaceBandwidth updates the bandwidth settings for all running sessions on the specified interface.
// This is called when bandwidth settings are saved via Config().Bandwidth().Save().
// It iterates all running sessions, and for each session on the specified interface:
// - Updates bandwidth based on UseGlobal setting
// - Saves the session (which triggers ApplyBandwidthUpdate via the save callback)
func (self *SessionsMgr) UpdateInterfaceBandwidth(ctx context.Context, ifname string, cfg sdkapi.IBandwdCfg) {
	log.Printf("[SessionsMgr] UpdateInterfaceBandwidth: updating sessions on interface %s", ifname)

	self.sessions.Range(func(key, value any) bool {
		rs := value.(*RunningSession)
		lan := rs.Lan()

		if lan == nil || lan.Name() != ifname {
			return true // Continue to next session
		}

		// Skip stopped sessions
		if rs.IsStopped() {
			log.Printf("[SessionsMgr] UpdateInterfaceBandwidth: skipping stopped session for device %d", rs.ClientId())
			return true
		}

		session := rs.GetSession()
		if session == nil {
			return true
		}

		// Determine bandwidth based on UseGlobal setting
		var downMbits, upMbits int
		if cfg.UseGlobal {
			downMbits = cfg.GlobalDownMbits
			upMbits = cfg.GlobalUpMbits
		} else {
			downMbits = cfg.UserDownMbits
			upMbits = cfg.UserUpMbits
		}

		log.Printf("[SessionsMgr] UpdateInterfaceBandwidth: updating session %d - Down=%d, Up=%d, UseGlobal=%v",
			session.ID(), downMbits, upMbits, cfg.UseGlobal)

		// Update session bandwidth settings
		session.SetData(sdkapi.SessionUpdateData{
			DownMbits:      &downMbits,
			UpMbits:        &upMbits,
			UseGlobalSpeed: &cfg.UseGlobal,
		})

		// Save triggers the save callback which calls ApplyBandwidthUpdate
		if err := session.Save(ctx); err != nil {
			log.Printf("[SessionsMgr] UpdateInterfaceBandwidth: failed to save session %d: %v", session.ID(), err)
			// Continue updating other sessions
		}

		return true // Continue to next session
	})

	log.Printf("[SessionsMgr] UpdateInterfaceBandwidth: completed for interface %s", ifname)
}

// FindClientById finds a client device by its database ID.
func (self *SessionsMgr) FindClientById(ctx context.Context, devId int64) (sdkapi.IClientDevice, error) {
	device, err := self.mdl.Device().Find(ctx, devId)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	clnt := NewClientDevice(self.db, self.mdl, device)
	clnt.SetIsConnectedFunc(func(deviceID int64) bool {
		return self.IsConnected(clnt)
	})

	return clnt, nil
}

// FindClientByMac finds a client device by its MAC address.
func (self *SessionsMgr) FindClientByMac(ctx context.Context, mac string) (sdkapi.IClientDevice, error) {
	device, err := self.mdl.Device().FindByMac(ctx, mac)
	if err != nil {
		return nil, fmt.Errorf("device not found by MAC %s: %w", mac, err)
	}

	clnt := NewClientDevice(self.db, self.mdl, device)
	clnt.SetIsConnectedFunc(func(deviceID int64) bool {
		return self.IsConnected(clnt)
	})

	return clnt, nil
}

// FindClientByIp finds a client device by its IP address.
func (self *SessionsMgr) FindClientByIp(ctx context.Context, ip string) (sdkapi.IClientDevice, error) {
	device, err := self.mdl.Device().FindByIp(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("device not found by IP %s: %w", ip, err)
	}

	clnt := NewClientDevice(self.db, self.mdl, device)
	clnt.SetIsConnectedFunc(func(deviceID int64) bool {
		return self.IsConnected(clnt)
	})

	return clnt, nil
}
