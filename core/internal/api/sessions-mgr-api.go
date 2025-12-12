package api

import (
	"context"

	"core/db/models"
	"core/internal/connmgr"
	sdkapi "sdk/api"

	"github.com/google/uuid"
)

func NewSessionsMgrApi(pluginApi *PluginApi) *SessionsMgrApi {
	sessionsMgrApi := &SessionsMgrApi{
		pluginApi: pluginApi,
	}
	pluginApi.SessionsMgrAPI = sessionsMgrApi
	return sessionsMgrApi
}

type SessionsMgrApi struct {
	pluginApi *PluginApi
}

// FindClientById finds a client device by its ID.
func (self *SessionsMgrApi) FindClientById(ctx context.Context, devId int64) (sdkapi.IClientDevice, error) {
	device, err := self.pluginApi.models.Device().Find(ctx, devId)
	if err != nil {
		return nil, err
	}
	clnt := connmgr.NewClientDevice(self.pluginApi.db, self.pluginApi.models, device)
	return clnt, nil
}

// FindDeviceByUUID finds a client device by its globally unique identifier.
func (self *SessionsMgrApi) FindDeviceByUUID(ctx context.Context, uuid string) (sdkapi.IClientDevice, error) {
	device, err := self.pluginApi.models.Device().FindByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	clnt := connmgr.NewClientDevice(self.pluginApi.db, self.pluginApi.models, device)
	return clnt, nil
}

// FindSessionByUUID finds a session by its globally unique identifier.
func (self *SessionsMgrApi) FindSessionByUUID(ctx context.Context, uuid string) (sdkapi.IClientSession, error) {
	session, err := self.pluginApi.models.Session().FindByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	cs := connmgr.NewClientSession(self.pluginApi.db, self.pluginApi.models, self.pluginApi.PluginsMgr(), session)
	return cs, nil
}

// Connect connects a client device to the internet.
func (self *SessionsMgrApi) Connect(ctx context.Context, clnt sdkapi.IClientDevice, notify string) error {
	return self.pluginApi.SessionMgr.Connect(ctx, clnt, notify)
}

// Disconnect disconnects a client device from the internet.
func (self *SessionsMgrApi) Disconnect(ctx context.Context, clnt sdkapi.IClientDevice, notify string) error {
	return self.pluginApi.SessionMgr.Disconnect(ctx, clnt, notify)
}

// IsConnected checks if a client device is connected to the internet.
func (self *SessionsMgrApi) IsConnected(clnt sdkapi.IClientDevice) bool {
	return self.pluginApi.SessionMgr.IsConnected(clnt)
}

// CreateSession creates a session for the client device using the plugin's package name.
func (self *SessionsMgrApi) CreateSession(ctx context.Context, params sdkapi.CreateSessionParams) (sdkapi.IClientSession, error) {
	sessionUUID := uuid.New().String()
	pkg := self.pluginApi.Info().Package
	session, err := self.pluginApi.models.Session().Create(ctx, models.CreateSessionParams{
		UUID:        sessionUUID,
		PluginPkg:   pkg,
		DeviceID:    params.DevId,
		SessionType: params.SessionType,
		TimeSecs:    params.TimeSecs,
		DataMbytes:  params.DataMbytes,
		ExpDays:     params.ExpDays,
		DownMbits:   params.DownMbits,
		UpMbits:     params.UpMbits,
		UseGlobal:   params.UseGlobal,
	})
	if err != nil {
		return nil, err
	}
	cs := connmgr.NewClientSession(self.pluginApi.db, self.pluginApi.models, self.pluginApi.PluginsMgr(), session)
	return cs, nil
}

// RunningSession returns the current running session of a client device.
func (self *SessionsMgrApi) RunningSession(clnt sdkapi.IClientDevice) (sdkapi.IClientSession, bool) {
	return self.pluginApi.SessionMgr.CurrSession(clnt)
}

// AvailableSession returns unconsumed session (if any) for the client device.
func (self *SessionsMgrApi) AvailableSession(ctx context.Context, clnt sdkapi.IClientDevice) (sdkapi.IClientSession, error) {
	return self.pluginApi.SessionMgr.GetSession(ctx, clnt)
}

// SessionSummary returns the session summary for the client device.
func (self *SessionsMgrApi) SessionSummary(ctx context.Context, clnt sdkapi.IClientDevice) (*sdkapi.ClientSessionSummary, error) {
	return self.pluginApi.SessionMgr.SessionSummary(ctx, clnt)
}

// FindSessionByID finds a session by its database ID and wraps it into an IClientSession object.
func (self *SessionsMgrApi) FindSessionByID(ctx context.Context, sessionID int64) (sdkapi.IClientSession, error) {
	return self.pluginApi.SessionMgr.FindSessionByID(ctx, sessionID)
}

// NewSession wraps session data into an IClientSession object without performing
// additional database queries.
func (self *SessionsMgrApi) NewSession(params sdkapi.NewSessionParams) sdkapi.IClientSession {
	return self.pluginApi.SessionMgr.NewSession(params)
}

// NewClientDevice wraps device data into an IClientDevice object without performing
// additional database queries.
func (self *SessionsMgrApi) NewClientDevice(params sdkapi.NewDeviceParams) sdkapi.IClientDevice {
	return self.pluginApi.SessionMgr.NewClientDevice(params)
}

// OnSessionEvent registers a callback for session events.
func (self *SessionsMgrApi) OnSessionEvent(event sdkapi.SessionEvent, callback func(data sdkapi.SessionEventData)) {
	self.pluginApi.SessionMgr.OnSessionEvent(event, callback)
}

// OnClientEvent registers a callback for client device events.
func (self *SessionsMgrApi) OnClientEvent(event sdkapi.ClientEvent, callback func(clnt sdkapi.IClientDevice)) {
	self.pluginApi.SessionMgr.OnClientEvent(event, callback)
}
