package connmgr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"core/db"
	"core/db/models"
	"core/internal/network"
	"core/internal/utils/nftables"
	sdkapi "sdk/api"
)

var (
	ErrSessionQuery = errors.New("Error in session query")
	ErrSessionEmpty = errors.New("Device has no more available sessions.")
)

func NewSessionsMgr(dtb *db.Database, mdl *models.Models) *SessionsMgr {
	sessionMgr := &SessionsMgr{
		db:                    dtb,
		mdl:                   mdl,
		sessions:              sync.Map{},
		sessionEventCallbacks: make(map[string][]func(data sdkapi.SessionEventData)),
		clientEventCallbacks:  make(map[string][]func(clnt sdkapi.IClientDevice)),
	}
	return sessionMgr
}

type SessionsMgr struct {
	coreAPI               sdkapi.IPluginApi
	pluginsMgr            sdkapi.IPluginsMgrApi
	db                    *db.Database
	mdl                   *models.Models
	sessions              sync.Map
	sessionEventCallbacks map[string][]func(data sdkapi.SessionEventData)
	clientEventCallbacks  map[string][]func(clnt sdkapi.IClientDevice)
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

	// Then reset all started_at fields to NULL
	err = self.db.Queries.ResetAllStartedAt(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset started_at fields: %w", err)
	}

	return nil
}

func (self *SessionsMgr) OnSessionEvent(event string, callback func(data sdkapi.SessionEventData)) {
	self.sessionEventCallbacks[event] = append(self.sessionEventCallbacks[event], callback)
}

func (self *SessionsMgr) OnClientEvent(event string, callback func(clnt sdkapi.IClientDevice)) {
	self.clientEventCallbacks[event] = append(self.clientEventCallbacks[event], callback)
}

func (self *SessionsMgr) emitSessionEvent(event sdkapi.SessionEvent, session sdkapi.IClientSession, device sdkapi.IClientDevice) {
	data := sdkapi.SessionEventData{
		Session: session,
		Device:  device,
	}
	if callbacks, exists := self.sessionEventCallbacks[string(event)]; exists {
		for _, callback := range callbacks {
			callback(data)
		}
	}
}

func (self *SessionsMgr) emitClientEvent(event sdkapi.SessionEvent, clnt sdkapi.IClientDevice) {
	if callbacks, exists := self.clientEventCallbacks[string(event)]; exists {
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
			errReturnCh <- errors.New("Device is already connected.")
			return
		}

		_, err := self.GetSession(ctx, clnt)
		if err != nil {
			errReturnCh <- ErrSessionEmpty
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
	v, ok := self.sessions.Load(clnt.Id())
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
				rs, err = NewRunningSession(clnt, cs)
				if err != nil {
					errCh <- err
					return
				}

				err = rs.Start(ctx, cs)
				if err != nil {
					errCh <- err
					return
				}

				self.sessions.Store(clnt.Id(), rs)
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
	v, ok := self.sessions.Load(clnt.Id())
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

		self.sessions.Delete(clnt.Id())

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

// SessionSummary
func (self *SessionsMgr) SessionSummary(ctx context.Context, clnt sdkapi.IClientDevice) (*sdkapi.ClientSessionSummary, error) {
	summary, err := self.mdl.Session().Summary(ctx, clnt.Id())
	if err != nil {
		return nil, err
	}

	rs, ok := self.getRunningSession(clnt)
	if !ok {
		return summary, nil
	}

	// Get unsaved data consumption diff
	mbDiff := rs.Diff()

	// Time is calculated dynamically in RemainingTime(), so use it directly
	remainingTime := rs.GetSession().RemainingTime()
	if remainingTime < 0 {
		remainingTime = 0
	}

	// Subtract unsaved data consumption from remaining data
	remainingData := summary.RemainingDataMbytes - mbDiff
	if remainingData < 0 {
		remainingData = 0
	}

	return &sdkapi.ClientSessionSummary{
		RemainingTimeSecs:   remainingTime,
		RemainingDataMbytes: remainingData,
	}, nil
}
