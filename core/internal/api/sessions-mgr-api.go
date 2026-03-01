package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"core/db/models"
	coreQueries "core/db/queries"
	sdkapi "sdk/api"
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
	return self.pluginApi.SessionMgr.FindDeviceByID(ctx, devId)
}

// FindDeviceByUUID finds a client device by its globally unique identifier.
func (self *SessionsMgrApi) FindDeviceByUUID(ctx context.Context, uuid string) (sdkapi.IClientDevice, error) {
	return self.pluginApi.SessionMgr.FindDeviceByUUID(ctx, uuid)
}

// FindSessionByUUID finds a session by its globally unique identifier.
func (self *SessionsMgrApi) FindSessionByUUID(ctx context.Context, uuid string) (sdkapi.IClientSession, error) {
	return self.pluginApi.SessionMgr.FindSessionByUUID(ctx, uuid)
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
	// Validate UUID
	if params.UUID == "" {
		return nil, errors.New("session UUID is required")
	}

	// Fetch device for event emission
	device, err := self.FindClientById(ctx, params.DevId)
	if err != nil {
		return nil, err
	}

	// Use provided UUID and get plugin package
	sessionUUID := params.UUID
	pkg := self.pluginApi.Info().Package

	// Create session in database
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

	// Wrap session in IClientSession interface with save callback
	cs := self.pluginApi.SessionMgr.NewClientSession(sdkapi.NewClientSessionParams{
		ID:              session.ID(),
		UUID:            session.UUID(),
		ProviderPkg:     session.ProviderPkg(),
		DeviceID:        session.DeviceID(),
		SessionType:     sdkapi.SessionType(session.SessionType()),
		TimeSecs:        session.TimeSecs(),
		DataMbytes:      session.DataMbyte(),
		ConsumptionSecs: session.TimeConsumed(),
		ConsumptionMb:   session.DataConsumed(),
		StartedAt:       session.StartedAt(),
		ResumedAt:       session.ResumedAt(),
		ExpDays:         session.ExpDays(),
		DownMbits:       session.DownMbits(),
		UpMbits:         session.UpMbits(),
		UseGlobal:       session.UseGlobal(),
		CreatedAt:       session.CreatedAt(),
		UpdatedAt:       session.UpdatedAt(),
	})

	// Set consumption values if provided (for cloud sync)
	if params.ConsumptionSecs > 0 || params.ConsumptionMb > 0 {
		cs.SetTimeCons(params.ConsumptionSecs)
		cs.SetDataCons(params.ConsumptionMb)
		if err = cs.Save(ctx); err != nil {
			return nil, err
		}
	}

	// Emit EventSessionCreated - notify plugins that session was created
	self.pluginApi.SessionMgr.EmitSessionEvent(sdkapi.EventSessionCreated, cs, device)

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

// NewClientSession wraps session data into an IClientSession object without performing
// additional database queries.
func (self *SessionsMgrApi) NewClientSession(params sdkapi.NewClientSessionParams) sdkapi.IClientSession {
	return self.pluginApi.SessionMgr.NewClientSession(params)
}

// NewClientDevice wraps device data into an IClientDevice object without performing
// additional database queries.
func (self *SessionsMgrApi) NewClientDevice(params sdkapi.NewDeviceParams) sdkapi.IClientDevice {
	return self.pluginApi.SessionMgr.NewClientDevice(params)
}

// OnSessionEvent registers a callback for session events.
func (self *SessionsMgrApi) OnSessionEvent(event sdkapi.SessionEvent, callback func(data sdkapi.SessionEventData) error) {
	self.pluginApi.SessionMgr.OnSessionEvent(event, callback)
}

// OnClientEvent registers a callback for client device events.
func (self *SessionsMgrApi) OnClientEvent(event sdkapi.ClientEvent, callback func(clnt sdkapi.IClientDevice) error) {
	self.pluginApi.SessionMgr.OnClientEvent(event, callback)
}

// ListSessions returns a paginated list of sessions with optional search and filters.
func (self *SessionsMgrApi) ListSessions(ctx context.Context, params sdkapi.ListSessionsParams) (sdkapi.ListSessionsResult, error) {
	q := coreQueries.New(self.pluginApi.db.DB)
	offset := int64(params.PerPage * (params.Page - 1))

	// Prepare search parameter
	var search interface{}
	if params.Search != nil && *params.Search != "" {
		search = *params.Search
	}

	// Prepare device ID parameter
	var deviceID interface{}
	if params.DeviceID != nil {
		deviceID = *params.DeviceID
	}

	// Prepare availability parameter (nil = all)
	availability := "all"
	if params.Availability != nil {
		availability = string(*params.Availability)
	}

	// Prepare session type parameter
	var sessionType interface{}
	if params.SessionType != nil {
		sessionType = string(*params.SessionType)
	}

	// Prepare date parameters
	var dateStart, dateEnd interface{}
	if params.DateStart != nil {
		dateStart = params.DateStart.Format("2006-01-02")
	}
	if params.DateEnd != nil {
		dateEnd = params.DateEnd.Format("2006-01-02")
	}

	// Prepare time/data filter parameters
	var timeSecsGt, timeSecsLt interface{}
	var dataMbGt, dataMbLt interface{}
	if params.TimeSecsGt != nil {
		timeSecsGt = *params.TimeSecsGt
	}
	if params.TimeSecsLt != nil {
		timeSecsLt = *params.TimeSecsLt
	}
	if params.DataMbGt != nil {
		dataMbGt = *params.DataMbGt
	}
	if params.DataMbLt != nil {
		dataMbLt = *params.DataMbLt
	}

	// payment type
	paymentType := "all"
	if params.PaymentType != nil {
		paymentType = string(*params.PaymentType)
	}

	paginatedRows, err := q.GetSessionsPaginated(ctx, coreQueries.GetSessionsPaginatedParams{
		Search:       search,
		DeviceID:     deviceID,
		Availability: availability,
		SessionType:  sessionType,
		DateStart:    dateStart,
		DateEnd:      dateEnd,
		TimeSecsGt:   timeSecsGt,
		TimeSecsLt:   timeSecsLt,
		DataMbGt:     dataMbGt,
		DataMbLt:     dataMbLt,
		RowLimit:     int64(params.PerPage),
		RowOffset:    offset,
		PaymentType:  paymentType,
	})
	if err != nil {
		return sdkapi.ListSessionsResult{}, fmt.Errorf("unable to list sessions: %w", err)
	}

	filteredRows, err := q.GetSessionsFiltered(ctx, coreQueries.GetSessionsFilteredParams{
		Search:       search,
		DeviceID:     deviceID,
		Availability: availability,
		SessionType:  sessionType,
		DateStart:    dateStart,
		DateEnd:      dateEnd,
		TimeSecsGt:   timeSecsGt,
		TimeSecsLt:   timeSecsLt,
		DataMbGt:     dataMbGt,
		DataMbLt:     dataMbLt,
		PaymentType:  paymentType,
	})
	if err != nil {
		return sdkapi.ListSessionsResult{}, fmt.Errorf("unable to count sessions: %w", err)
	}

	return sdkapi.ListSessionsResult{
		PaginatedSessions: self.wrapManySessions(paginatedRows),
		FilteredSessions:  self.wrapManySessions(filteredRows),
	}, nil
}

// wrapManySessions wraps multiple session rows into IClientSession objects.
func (self *SessionsMgrApi) wrapManySessions(rows []coreQueries.Session) []sdkapi.IClientSession {
	sessions := make([]sdkapi.IClientSession, len(rows))
	for i, row := range rows {
		sessions[i] = self.wrapSession(row)
	}
	return sessions
}

// wrapSession wraps a single session row into an IClientSession object.
func (self *SessionsMgrApi) wrapSession(row coreQueries.Session) sdkapi.IClientSession {
	var startedAt, resumedAt *time.Time
	if row.StartedAt.Valid {
		startedAt = &row.StartedAt.Time
	}
	if row.ResumedAt.Valid {
		resumedAt = &row.ResumedAt.Time
	}

	var expDays *int
	if row.ExpDays.Valid {
		d := int(row.ExpDays.Int64)
		expDays = &d
	}

	return self.pluginApi.SessionMgr.NewClientSession(sdkapi.NewClientSessionParams{
		ID:              row.ID,
		UUID:            row.Uuid,
		ProviderPkg:     row.ProviderPkg,
		DeviceID:        row.DeviceID,
		SessionType:     sdkapi.SessionType(row.SessionType),
		TimeSecs:        int(row.TimeSecs),
		DataMbytes:      row.DataMbytes,
		ConsumptionSecs: int(row.ConsumptionSecs),
		ConsumptionMb:   row.ConsumptionMb,
		StartedAt:       startedAt,
		ResumedAt:       resumedAt,
		ExpDays:         expDays,
		DownMbits:       int(row.DownMbits),
		UpMbits:         int(row.UpMbits),
		UseGlobal:       row.UseGlobal,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	})
}

// DeleteSession deletes a session by ID. If the session is currently running,
// it disconnects the device first. Emits EventSessionDeleted after deletion.
func (self *SessionsMgrApi) DeleteSession(ctx context.Context, sessionID int64) error {
	// Find the session first to get its data (including UUID for cloud sync)
	session, err := self.FindSessionByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Get the device for the event
	device, err := self.FindClientById(ctx, session.DeviceID())
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	// If session is running, disconnect first
	if session.IsRunning() && self.IsConnected(device) {
		if disconnectErr := self.Disconnect(ctx, device, "Session deleted"); disconnectErr != nil {
			// Log but don't fail - we still want to delete
			self.pluginApi.Logger().Error(fmt.Sprintf("Failed to disconnect device before session deletion: %v", disconnectErr))
		}
	}

	// Delete from local database
	if err := self.pluginApi.models.Session().Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Emit EventSessionDeleted AFTER deletion
	// The session object still has all the data needed (UUID, etc.)
	self.pluginApi.SessionMgr.EmitSessionEvent(sdkapi.EventSessionDeleted, session, device)

	return nil
}
