package connmgr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"core/db"
	"core/db/models"
	"core/internal/network"
	"core/internal/utils/nftables"
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
		sessionEventCallbacks: make(map[sdkapi.SessionEvent][]func(data sdkapi.SessionEventData)),
		clientEventCallbacks:  make(map[sdkapi.ClientEvent][]func(clnt sdkapi.IClientDevice)),
	}
	return sessionMgr
}

type SessionsMgr struct {
	coreAPI               sdkapi.IPluginApi
	pluginsMgr            sdkapi.IPluginsMgrApi
	db                    *db.Database
	mdl                   *models.Models
	sessions              sync.Map
	sessionEventCallbacks map[sdkapi.SessionEvent][]func(data sdkapi.SessionEventData)
	clientEventCallbacks  map[sdkapi.ClientEvent][]func(clnt sdkapi.IClientDevice)
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

func (self *SessionsMgr) OnSessionEvent(event sdkapi.SessionEvent, callback func(data sdkapi.SessionEventData)) {
	self.sessionEventCallbacks[event] = append(self.sessionEventCallbacks[event], callback)
}

func (self *SessionsMgr) OnClientEvent(event sdkapi.ClientEvent, callback func(clnt sdkapi.IClientDevice)) {
	self.clientEventCallbacks[event] = append(self.clientEventCallbacks[event], callback)
}

func (self *SessionsMgr) emitSessionEvent(event sdkapi.SessionEvent, session sdkapi.IClientSession, device sdkapi.IClientDevice) {
	data := sdkapi.SessionEventData{
		Session: session,
		Device:  device,
	}
	if callbacks, exists := self.sessionEventCallbacks[event]; exists {
		for _, callback := range callbacks {
			callback(data)
		}
	}
}

func (self *SessionsMgr) emitClientEvent(event sdkapi.ClientEvent, clnt sdkapi.IClientDevice) {
	if callbacks, exists := self.clientEventCallbacks[event]; exists {
		for _, callback := range callbacks {
			callback(clnt)
		}
	}
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
		err := nftables.Disconnect(rs.mac, reason)
		if err != nil {
		}

		lan, err := network.FindByIp(rs.ip)
		if err != nil {
		}

		if lan.Name() == iface {
			rs.Stop(ctx)
		}

		return true
	})
}

func (self *SessionsMgr) Connect(ctx context.Context, clnt sdkapi.IClientDevice, notify string) error {
	errReturnCh := make(chan error)

	if clnt.Status() == sdkapi.Blocked {
		return errors.New(self.coreAPI.Translate("error", "Device is blocked"))
	}

	go func() {
		if _, ok := self.CurrSession(clnt); ok {
			errReturnCh <- errors.New(self.coreAPI.Translate("error", "Device is already connected"))
			return
		}

		_, err := self.GetSession(ctx, clnt)
		if err != nil {
			errReturnCh <- errors.New(self.coreAPI.Translate("error", "Device has no more available sessions"))
			return
		}

		if !nftables.IsConnected(clnt.MacAddr()) {
			if err := nftables.Connect(clnt.IpAddr(), clnt.MacAddr()); err != nil {
				errReturnCh <- err
				return
			}
		} else {
		}

		startCh := make(chan error)
		go self.loopSessions(startCh, clnt)

		err = <-startCh
		close(startCh)

		if err != nil {
			errReturnCh <- err
			return
		}

		clnt.Emit(string(sdkapi.EventSessionConnected), []byte(notify))
		if session, ok := self.CurrSession(clnt); ok {
			self.emitSessionEvent(sdkapi.EventSessionConnected, session, clnt)
		}
		self.emitClientEvent(sdkapi.EventClientConnected, clnt)
		errReturnCh <- nil
	}()

	// Handle error from goroutine
	err := <-errReturnCh
	if err == nil {
		err = clnt.Update(ctx, sdkapi.UpdateDeviceParams{
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
	err := self.endSession(ctx, clnt)
	if err != nil {
		return err
	}

	clnt.Emit(string(sdkapi.EventSessionDisconnected), []byte(notify))
	if session, ok := self.CurrSession(clnt); ok {
		self.emitSessionEvent(sdkapi.EventSessionDisconnected, session, clnt)
	}
	self.emitClientEvent(sdkapi.EventClientDisconnected, clnt)

	return clnt.Update(ctx, sdkapi.UpdateDeviceParams{
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

	return rs.session, true
}

func (self *SessionsMgr) loopSessions(resultCh chan<- error, clnt sdkapi.IClientDevice) {
	var callbackDone atomic.Bool
	ctx := context.Background()

	for nftables.IsConnected(clnt.MacAddr()) {
		errCh := make(chan error)

		go func() {
			cs, err := self.GetSession(ctx, clnt)
			if err != nil {
				errCh <- err
				return
			}

			rs, ok := self.getRunningSession(clnt)
			if !ok {
				rs, err = NewRunningSession(clnt, cs, self)
				if err != nil {
					errCh <- err
					return
				}

				err = rs.Start(ctx, cs)
				if err != nil {
					errCh <- err
					return
				}

				self.sessions.Store(clnt.ID(), rs)
			} else {
				err = rs.Start(ctx, cs)
				if err != nil {
					errCh <- err
					return
				}
			}

			// Start was successful
			if !callbackDone.Load() {
				resultCh <- nil
				callbackDone.Store(true)
			}

			err = <-rs.Done()
			errCh <- err
		}()

		err := <-errCh

		if !callbackDone.Load() {
			resultCh <- err
			callbackDone.Store(true)
		}

		if err != nil {
			self.Disconnect(ctx, clnt, err.Error())
			return
		}
	}
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
	errCh := make(chan error)

	go func() {
		if nftables.IsConnected(clnt.MacAddr()) {
			err := nftables.Disconnect(clnt.IpAddr(), clnt.MacAddr())
			if err != nil {
				errCh <- err
				return
			}
		}

		rs, ok := self.getRunningSession(clnt)

		if ok {
			err := rs.Stop(ctx)
			if err != nil {
				errCh <- err
				return
			}

			err = rs.CleanupTc()
			if err != nil {
				errCh <- err
				return
			}
		}

		self.sessions.Delete(clnt.ID())

		errCh <- nil
	}()

	return <-errCh
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
