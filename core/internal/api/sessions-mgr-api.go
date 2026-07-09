package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"core/db/models"
	coreQueries "core/db/queries"
	"core/internal/sessmgr"
	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
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

	// Give subscribers a chance to veto creation before the INSERT. The session is an
	// in-memory preview (ID == 0), so cancelling requires no rollback.
	preview := self.pluginApi.SessionMgr.NewClientSession(newClientSessionPreviewParams(pkg, params))
	if err := self.pluginApi.EventsMgr.EmitSessionEvent(ctx, sdkapi.EventSessionBeforeCreate, sdkapi.SessionEventData{Session: preview}); err != nil {
		return nil, err
	}

	cs, err := self.insertSession(ctx, nil, sessionUUID, pkg, params)
	if err != nil {
		return nil, err
	}

	if err := self.persistSessionConsumption(ctx, cs, params); err != nil {
		return nil, err
	}

	// Emit EventSessionCreated - notify plugins that session was created
	self.pluginApi.EventsMgr.EmitSessionEvent(ctx, sdkapi.EventSessionCreated, sdkapi.SessionEventData{Session: cs})

	return cs, nil
}

// CreateSessions creates a batch of sessions. It emits EventSessionBatchBeforeCreate
// ONCE before any DB writes — a callback error cancels the whole batch, so no
// rollback is needed — then inserts every session inside a single database
// transaction (same as DeviceModel.Create/BatchRegisterClient), so a failure
// partway through automatically rolls back every insert made so far; no manual
// cleanup is needed. Only once the transaction commits does it persist any
// per-session consumption values, emit the per-session EventSessionCreated for
// each session, and finally emit EventSessionBatchCreated once with the full
// list — this ordering guarantees no subscriber ever observes a "created"
// session that a later failure in the same batch then rolls back.
func (self *SessionsMgrApi) CreateSessions(ctx context.Context, paramsList []sdkapi.CreateSessionParams) ([]sdkapi.IClientSession, error) {
	if len(paramsList) == 0 {
		return nil, nil
	}

	pkg := self.pluginApi.Info().Package
	previews := make([]sdkapi.IClientSession, 0, len(paramsList))
	for _, params := range paramsList {
		if params.UUID == "" {
			return nil, errors.New("session UUID is required")
		}
		previews = append(previews, self.pluginApi.SessionMgr.NewClientSession(newClientSessionPreviewParams(pkg, params)))
	}

	// Batch-level before-create event: fires once before any DB writes. An error
	// here cancels the whole batch with no rollback needed.
	if err := self.pluginApi.EventsMgr.EmitSessionBatchEvent(ctx, sdkapi.EventSessionBatchBeforeCreate, previews); err != nil {
		return nil, err
	}

	created := make([]sdkapi.IClientSession, 0, len(paramsList))
	err := sdkutils.RunInTx(self.pluginApi.db.DB, ctx, func(tx *sql.Tx) error {
		for _, params := range paramsList {
			cs, err := self.insertSession(ctx, tx, params.UUID, pkg, params)
			if err != nil {
				return err
			}
			created = append(created, cs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Persist any caller-supplied consumption values now that every insert has
	// committed. This runs outside the transaction — it goes through the
	// session's own save path on the pooled connection, not tx, so doing it
	// while tx was still open would contend for SQLite's single write lock.
	// A failure here can no longer be rolled back by the transaction (it already
	// committed), so compensate by deleting every session created in this batch
	// and returning an error — this keeps the batch all-or-nothing and guarantees
	// EventSessionCreated/EventSessionBatchCreated are only ever emitted once
	// every session in the batch (row + consumption) is fully persisted.
	for i, cs := range created {
		if err := self.persistSessionConsumption(ctx, cs, paramsList[i]); err != nil {
			self.rollbackCreatedSessions(ctx, created)
			return nil, fmt.Errorf("persist consumption for session %s: %w", cs.UUID(), err)
		}
	}

	for _, cs := range created {
		self.pluginApi.EventsMgr.EmitSessionEvent(ctx, sdkapi.EventSessionCreated, sdkapi.SessionEventData{Session: cs})
	}
	self.pluginApi.EventsMgr.EmitSessionBatchEvent(ctx, sdkapi.EventSessionBatchCreated, created)

	return created, nil
}

// insertSession inserts a single session row (optionally within tx — see
// SessionModel.Create) and wraps it into an IClientSession. It does NOT persist
// consumption values or emit EventSessionCreated — callers handle both once they
// know the row (or, for a batch, the whole transaction) has committed.
func (self *SessionsMgrApi) insertSession(ctx context.Context, tx *sql.Tx, sessionUUID, pkg string, params sdkapi.CreateSessionParams) (sdkapi.IClientSession, error) {
	session, err := self.pluginApi.models.Session().Create(ctx, tx, models.CreateSessionParams{
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

	return cs, nil
}

// rollbackCreatedSessions deletes sessions from a CreateSessions batch whose
// transaction already committed but a later, non-transactional step (e.g.
// persisting consumption values) then failed. Best-effort: there is no
// surrounding transaction left to roll back into, so a delete failure is
// logged rather than returned — the original error is what the caller needs
// to see, and it already takes precedence.
func (self *SessionsMgrApi) rollbackCreatedSessions(ctx context.Context, created []sdkapi.IClientSession) {
	for _, cs := range created {
		if err := self.pluginApi.models.Session().Delete(ctx, cs.ID()); err != nil {
			self.pluginApi.Logger().Error(fmt.Sprintf("rollback session %s after batch create failure: %v", cs.UUID(), err))
		}
	}
}

// newClientSessionPreviewParams maps caller-supplied CreateSessionParams into
// an in-memory NewClientSessionParams preview (ID left at 0) for the
// before-create events — shared by the single and batch creation paths so
// the two previews can never drift.
func newClientSessionPreviewParams(pkg string, params sdkapi.CreateSessionParams) sdkapi.NewClientSessionParams {
	return sdkapi.NewClientSessionParams{
		UUID:           params.UUID,
		ProviderPkg:    pkg,
		DeviceID:       params.DevId,
		Type:           params.Type,
		TimeSecs:       params.TimeSecs,
		DataMb:         params.DataMb,
		TimeCons:       params.TimeCons,
		DataCons:       params.DataCons,
		ExpDays:        params.ExpDays,
		DownMbits:      params.DownMbits,
		UpMbits:        params.UpMbits,
		UseGlobalSpeed: params.UseGlobalSpeed,
	}
}

// persistSessionConsumption sets and saves the caller-supplied initial
// consumption values (for cloud-sync imports), if any were provided. Uses
// PersistToDB to avoid triggering EventSessionChanged during creation.
func (self *SessionsMgrApi) persistSessionConsumption(ctx context.Context, cs sdkapi.IClientSession, params sdkapi.CreateSessionParams) error {
	if params.TimeCons > 0 || params.DataCons > 0 {
		cs.SetData(sdkapi.SessionUpdateData{
			TimeCons: &params.TimeCons,
			DataCons: &params.DataCons,
		})
		// Type assert to access internal PersistToDB method
		if internal, ok := cs.(*sessmgr.ClientSession); ok {
			if err := internal.PersistToDB(ctx); err != nil {
				return err
			}
		} else {
			// Fallback - should never happen in practice
			if err := cs.Save(ctx, nil); err != nil {
				return err
			}
		}
	}

	return nil
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
// it disconnects the device first. Emits EventSessionBeforeDelete first (a callback
// error cancels the deletion) and EventSessionDeleted after deletion.
func (self *SessionsMgrApi) DeleteSession(ctx context.Context, sessionID int64) error {
	// Find the session first to get its data (including UUID for cloud sync)
	session, err := self.FindSessionByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Give subscribers a chance to veto before any disconnect or deletion.
	if err := self.pluginApi.EventsMgr.EmitSessionEvent(ctx, sdkapi.EventSessionBeforeDelete, sdkapi.SessionEventData{Session: session}); err != nil {
		return err
	}

	return self.deleteSessionInternal(ctx, session)
}

// DeleteSessions deletes a batch of sessions by ID. It emits EventSessionBatchBeforeDelete
// ONCE before any deletion — a callback error cancels the whole batch — then deletes each
// session (emitting the per-session EventSessionDeleted). The single-session
// EventSessionBeforeDelete is intentionally NOT fired per item; the batch hook is the
// cancellation point for bulk deletes.
func (self *SessionsMgrApi) DeleteSessions(ctx context.Context, sessionIDs []int64) error {
	if len(sessionIDs) == 0 {
		return nil
	}

	// Resolve all sessions up front so the batch preview and the delete loop see the
	// same set; a missing session fails the whole batch before anything is removed.
	sessions := make([]sdkapi.IClientSession, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		session, err := self.FindSessionByID(ctx, id)
		if err != nil {
			return fmt.Errorf("session %d not found: %w", id, err)
		}
		sessions = append(sessions, session)
	}

	// Give subscribers a chance to veto the whole batch before any deletion.
	if err := self.pluginApi.EventsMgr.EmitSessionBatchEvent(ctx, sdkapi.EventSessionBatchBeforeDelete, sessions); err != nil {
		return err
	}

	for _, session := range sessions {
		if err := self.deleteSessionInternal(ctx, session); err != nil {
			return err
		}
	}
	return nil
}

// deleteSessionInternal disconnects the owning device (if connected) and deletes the
// session row, emitting EventSessionDeleted afterwards. It does NOT fire any "before"
// event — callers decide whether to fire the single or batch pre-delete hook.
func (self *SessionsMgrApi) deleteSessionInternal(ctx context.Context, session sdkapi.IClientSession) error {
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
		}
	}

	// Delete from local database
	if err := self.pluginApi.models.Session().Delete(ctx, session.ID()); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Emit EventSessionDeleted AFTER deletion
	// The session object still has all the data needed (UUID, etc.)
	self.pluginApi.EventsMgr.EmitSessionEvent(ctx, sdkapi.EventSessionDeleted, sdkapi.SessionEventData{Session: session})

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
