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
		db:                    dtb,
		mdl:                   mdl,
		sessions:              sync.Map{},
		sessionEventCallbacks: sync.Map{},
		clientEventCallbacks:  sync.Map{},
	}
	return sessionMgr
}

type SessionsMgr struct {
	coreAPI               sdkapi.IPluginApi
	pluginsMgr            sdkapi.IPluginsMgrApi
	db                    *db.Database
	mdl                   *models.Models
	sessions              sync.Map
	sessionEventCallbacks sync.Map // map[sdkapi.SessionEvent][]func(data sdkapi.SessionEventData) error
	clientEventCallbacks  sync.Map // map[sdkapi.ClientEvent][]func(clnt sdkapi.IClientDevice) error
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
	callbacks := []func(data sdkapi.SessionEventData) error{}
	if existing, ok := self.sessionEventCallbacks.Load(event); ok {
		callbacks = existing.([]func(data sdkapi.SessionEventData) error)
	}
	callbacks = append(callbacks, callback)
	self.sessionEventCallbacks.Store(event, callbacks)
}

func (self *SessionsMgr) OnClientEvent(event sdkapi.ClientEvent, callback func(clnt sdkapi.IClientDevice) error) {
	callbacks := []func(clnt sdkapi.IClientDevice) error{}
	if existing, ok := self.clientEventCallbacks.Load(event); ok {
		callbacks = existing.([]func(clnt sdkapi.IClientDevice) error)
	}
	callbacks = append(callbacks, callback)
	self.clientEventCallbacks.Store(event, callbacks)
}

func (self *SessionsMgr) emitSessionEvent(event sdkapi.SessionEvent, session sdkapi.IClientSession, device sdkapi.IClientDevice) error {
	data := sdkapi.SessionEventData{
		Session: session,
		Device:  device,
	}
	if callbacksVal, exists := self.sessionEventCallbacks.Load(event); exists {
		callbacks := callbacksVal.([]func(data sdkapi.SessionEventData) error)
		for _, callback := range callbacks {
			if err := callback(data); err != nil {
				return err
			}
		}
	}
	return nil
}

// EmitSessionEvent is a public wrapper for emitSessionEvent to allow API layer to trigger events
func (self *SessionsMgr) EmitSessionEvent(event sdkapi.SessionEvent, session sdkapi.IClientSession, device sdkapi.IClientDevice) error {
	return self.emitSessionEvent(event, session, device)
}

func (self *SessionsMgr) emitClientEvent(event sdkapi.ClientEvent, clnt sdkapi.IClientDevice) error {
	if callbacksVal, exists := self.clientEventCallbacks.Load(event); exists {
		callbacks := callbacksVal.([]func(clnt sdkapi.IClientDevice) error)
		for _, callback := range callbacks {
			if err := callback(clnt); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *SessionsMgr) ListenTraffic(trfk *network.TrafficMgr) {
	go func() {
		for data := range trfk.Listen() {
			go func(data *sdkapi.TrafficData) {
				self.sessions.Range(func(key, value any) bool {
					rs := value.(*RunningSession)
					rs.UpdateDataConsumption(data)
					return true
				})
			}(&data)
		}
	}()
}

func (self *SessionsMgr) ReloadSessions(ctx context.Context, iface string) error {
	errCh := make(chan error)

	go func() {
		self.sessions.Range(func(key, value any) bool {
			rs := value.(*RunningSession)
			lan := rs.Lan()

			if lan.Name() == iface {
				cs := rs.GetSession()
				err := cs.Reload(ctx)
				if err != nil {
					errCh <- err
					return false
				}

				err = rs.Start(ctx, cs)
				if err != nil {
					errCh <- err
					return false
				}
			}

			return true
		})

		errCh <- nil
	}()

	return <-errCh
}

func (self *SessionsMgr) StopSessions(ctx context.Context, iface string, reason string) {
	self.sessions.Range(func(key, value any) bool {
		rs := value.(*RunningSession)

		// Get values under lock to avoid data race
		ip := rs.IpAddr()
		mac := rs.MacAddr()

		lan, err := network.FindByIp(ip)
		if err != nil {
			log.Printf("[SessionsMgr] StopSessions: failed to find LAN for IP %s: %v", ip, err)
			return true
		}

		if lan.Name() == iface {
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
	if clnt.Status() == sdkapi.Blocked {
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
			Status:   int(sdkapi.Connected),
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
		Status:   int(sdkapi.Disconnected),
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
			if errors.Is(err, ErrSessionExpired) {
				// Session expired normally - reset state and continue to try next session
				// TC class/filter are preserved for reuse with the next session
				log.Printf("Session expired for device %s, checking for next available session...", clnt.MacAddr())
				rs.Reset()
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

	localSrc := NewClientSession(self.db, self.mdl, self.coreAPI.PluginsMgr(), s)
	return localSrc, nil
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

	// Calculate elapsed time for the running session since resumed_at
	var elapsedSecs int = 0
	if rs.GetSession().ResumedAt() != nil {
		elapsedSecs = int(time.Since(*rs.GetSession().ResumedAt()).Seconds())
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

// FindSessionByID finds a session by its database ID and wraps it into an IClientSession object.
func (self *SessionsMgr) FindSessionByID(ctx context.Context, sessionID int64) (sdkapi.IClientSession, error) {
	s, err := self.mdl.Session().Find(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	localSrc := NewClientSession(self.db, self.mdl, self.coreAPI.PluginsMgr(), s)
	return localSrc, nil
}

// NewSession wraps session data into an IClientSession object without performing
// additional database queries.
func (self *SessionsMgr) NewSession(params sdkapi.NewSessionParams) sdkapi.IClientSession {
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
	return NewClientSession(self.db, self.mdl, self.coreAPI.PluginsMgr(), s)
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
	return NewClientDevice(self.db, self.mdl, d)
}
