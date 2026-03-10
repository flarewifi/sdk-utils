package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"core/db/models"
	coreQueries "core/db/queries"
	"core/internal/sessmgr"
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

// FindClientByMac finds a client device by its MAC address.
func (self *SessionsMgrApi) FindClientByMac(ctx context.Context, mac string) (sdkapi.IClientDevice, error) {
	return self.pluginApi.SessionMgr.FindClientByMac(ctx, mac)
}

// FindClientByIp finds a client device by its IP address.
func (self *SessionsMgrApi) FindClientByIp(ctx context.Context, ip string) (sdkapi.IClientDevice, error) {
	return self.pluginApi.SessionMgr.FindClientByIp(ctx, ip)
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

	// Use provided UUID and get plugin package
	sessionUUID := params.UUID
	pkg := self.pluginApi.Info().Package

	// Create session in database
	session, err := self.pluginApi.models.Session().Create(ctx, models.CreateSessionParams{
		UUID:           sessionUUID,
		PluginPkg:      pkg,
		DeviceID:       params.DevId,
		Type:           params.Type,
		TimeSecs:       params.TimeSecs,
		DataMb:         params.DataMb,
		ExpDays:        params.ExpDays,
		DownMbits:      params.DownMbits,
		UpMbits:        params.UpMbits,
		UseGlobalSpeed: params.UseGlobalSpeed,
	})
	if err != nil {
		return nil, err
	}

	// Wrap session in IClientSession interface with save callback
	cs := self.pluginApi.SessionMgr.NewClientSession(sdkapi.NewClientSessionParams{
		ID:             session.ID(),
		UUID:           session.UUID(),
		ProviderPkg:    session.ProviderPkg(),
		DeviceID:       session.DeviceID(),
		Type:           sdkapi.SessionType(session.SessionType()),
		TimeSecs:       session.TimeSecs(),
		DataMb:         session.DataMbyte(),
		TimeCons:       session.TimeConsumed(),
		DataCons:       session.DataConsumed(),
		StartedAt:      session.StartedAt(),
		ResumedAt:      session.ResumedAt(),
		ExpDays:        session.ExpDays(),
		DownMbits:      session.DownMbits(),
		UpMbits:        session.UpMbits(),
		UseGlobalSpeed: session.UseGlobal(),
		CreatedAt:      session.CreatedAt(),
		UpdatedAt:      session.UpdatedAt(),
	})

	// Set consumption values if provided (for cloud sync)
	// Use PersistToDB to avoid triggering EventSessionChanged during creation
	if params.TimeCons > 0 || params.DataCons > 0 {
		cs.SetData(sdkapi.SessionUpdateData{
			TimeCons: &params.TimeCons,
			DataCons: &params.DataCons,
		})
		// Type assert to access internal PersistToDB method
		if internal, ok := cs.(*sessmgr.ClientSession); ok {
			if err = internal.PersistToDB(ctx); err != nil {
				return nil, err
			}
		} else {
			// Fallback - should never happen in practice
			if err = cs.Save(ctx, nil); err != nil {
				return nil, err
			}
		}
	}

	// Emit EventSessionCreated - notify plugins that session was created
	self.pluginApi.EventsMgr.EmitSessionEvent(sdkapi.EventSessionCreated, sdkapi.SessionEventData{Session: cs})

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

// OnSessionEvent registers a callback for session events (delegates to global EventsManager).
func (self *SessionsMgrApi) OnSessionEvent(event sdkapi.SessionEvent, callback func(data sdkapi.SessionEventData) error) {
	self.pluginApi.EventsMgr.OnSessionEvent(event, callback)
}

// OnClientEvent registers a callback for client device events (delegates to global EventsManager).
func (self *SessionsMgrApi) OnClientEvent(event sdkapi.ClientEvent, callback func(clnt sdkapi.IClientDevice) error) {
	self.pluginApi.EventsMgr.OnClientEvent(event, callback)
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
		ID:             row.ID,
		UUID:           row.Uuid,
		ProviderPkg:    row.ProviderPkg,
		DeviceID:       row.DeviceID,
		Type:           sdkapi.SessionType(row.SessionType),
		TimeSecs:       int(row.TimeSecs),
		DataMb:         row.DataMbytes,
		TimeCons:       int(row.ConsumptionSecs),
		DataCons:       row.ConsumptionMb,
		StartedAt:      startedAt,
		ResumedAt:      resumedAt,
		ExpDays:        expDays,
		DownMbits:      int(row.DownMbits),
		UpMbits:        int(row.UpMbits),
		UseGlobalSpeed: row.UseGlobal,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
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

	// Disconnect the device if it is currently connected.
	// Do NOT pre-check session.IsRunning() first — that check races with the timer goroutine
	// (StopWithReason), which can stop the session between the check and the Disconnect call,
	// causing duplicate nftables calls and spurious error logs.
	// endSession() and StopWithReason() are both idempotent (stopped guard), so it is safe
	// to call Disconnect unconditionally; it becomes a no-op when already disconnected.
	if self.IsConnected(device) {
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
	self.pluginApi.EventsMgr.EmitSessionEvent(sdkapi.EventSessionDeleted, sdkapi.SessionEventData{Session: session})

	return nil
}

// ListRunningSessions returns all currently active (running) sessions.
func (self *SessionsMgrApi) ListRunningSessions() ([]sdkapi.IClientSession, error) {
	return self.pluginApi.SessionMgr.ListRunningSessions()
}

// FindRunningSessionByUUID finds a currently running session by its UUID.
// Returns the session and true if found, or nil and false if no running session
// exists with the given UUID.
func (self *SessionsMgrApi) FindRunningSessionByUUID(uuid string) (sdkapi.IClientSession, bool) {
	return self.pluginApi.SessionMgr.FindRunningSessionByUUID(uuid)
}

// MergeClientDevices merges the source device into the target device.
func (self *SessionsMgrApi) MergeClientDevices(ctx context.Context, targetID, sourceID int64) error {
	return self.pluginApi.SessionMgr.MergeClientDevices(ctx, targetID, sourceID)
}
