/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"context"
	"time"
)

// Session Type represents the type of a client session.
type SessionType string

const (
	SessionTypeTime       SessionType = "time"
	SessionTypeData       SessionType = "data"
	SessionTypeTimeOrData SessionType = "time-or-data"
)

// ClientSessionSummary represents a summary of a client's session.
type ClientSessionSummary struct {
	RemainingTimeSecs int
	RemainingDataMb   float64
}

// CreateSessionParams holds parameters for creating a new client session.
type CreateSessionParams struct {
	UUID           string // Required: Session UUID (use sdkutils.NewUUID() to generate)
	DevId          int64
	Type           SessionType
	TimeSecs       int
	DataMb         float64
	ExpDays        *int
	DownMbits      int
	UpMbits        int
	UseGlobalSpeed bool
	TimeCons       int     // Optional: Time consumption in seconds
	DataCons       float64 // Optional: Data consumption in megabytes
}

// NewClientSessionParams holds session data fields for wrapping an existing session row
// into an IClientSession object without performing database queries.
type NewClientSessionParams struct {
	ID             int64
	UUID           string
	ProviderPkg    string
	DeviceID       int64
	Type           SessionType
	TimeSecs       int
	DataMb         float64
	TimeCons       int
	DataCons       float64
	StartedAt      *time.Time
	ResumedAt      *time.Time
	ExpDays        *int
	DownMbits      int
	UpMbits        int
	UseGlobalSpeed bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewDeviceParams holds device data fields for wrapping an existing device row
// into an IClientDevice object without performing database queries.
type NewDeviceParams struct {
	ID          int64
	UUID        string
	CookieToken string
	MacAddress  string
	Ipv4Address string
	Ipv6Address string
	Hostname    string
	Status      DeviceStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ISessionsMgrApi is used to manage client devices.
type ISessionsMgrApi interface {

	// Finds a client device by its ID.
	//
	// Deprecated: use ClientsMgr().FindClientById() instead. This method is kept here
	// for backward compatibility and delegates to the same underlying implementation.
	FindClientById(ctx context.Context, devId int64) (IClientDevice, error)

	// Finds a client by MAC address
	//
	// Deprecated: use ClientsMgr().FindClientByMac() instead. This method is kept here
	// for backward compatibility and delegates to the same underlying implementation.
	FindClientByMac(ctx context.Context, mac string) (IClientDevice, error)

	// FindClientByIp finds a client device by its IP address.
	// This is useful for scenarios where you have an IP address (e.g., from an HTTP request)
	// and need to find the associated device.
	//
	// Deprecated: use ClientsMgr().FindClientByIp() instead. This method is kept here
	// for backward compatibility and delegates to the same underlying implementation.
	FindClientByIp(ctx context.Context, ip string) (IClientDevice, error)

	// FindDeviceByUUID finds a client device by its globally unique identifier.
	// This is useful for referencing devices by their UUID rather than local database ID.
	//
	// Deprecated: use ClientsMgr().FindClientByUUID() instead. This method is kept here
	// for backward compatibility and delegates to the same underlying implementation.
	FindDeviceByUUID(ctx context.Context, uuid string) (IClientDevice, error)

	// FindSessionByUUID finds a session by its globally unique identifier.
	// This is useful for querying or terminating sessions by their UUID.
	FindSessionByUUID(ctx context.Context, uuid string) (IClientSession, error)

	// Connects a client device to the internet.
	// Note: ctx is accepted for API compatibility but ignored internally.
	Connect(ctx context.Context, clnt IClientDevice, notify string) error

	// Disconnects a client device from the internet.
	// If notify is not nil, then the client device will be notified of the disconnection.
	// Note: ctx is accepted for API compatibility but ignored internally.
	Disconnect(ctx context.Context, clnt IClientDevice, notify string) error

	// Checks if a client device is connected to the internet.
	IsConnected(clnt IClientDevice) (connected bool)

	// Create a session for the client device
	CreateSession(ctx context.Context, params CreateSessionParams) (IClientSession, error)

	// CreateSessions creates a batch of sessions in one call. It emits
	// EventSessionBatchBeforeCreate ONCE before any DB writes — a callback error
	// cancels the whole batch — then creates each session (emitting the per-session
	// EventSessionCreated), and finally emits EventSessionBatchCreated once with the
	// full list. The single-session EventSessionBeforeCreate is not fired per item;
	// the batch hook is the cancellation point for bulk creates.
	CreateSessions(ctx context.Context, paramsList []CreateSessionParams) ([]IClientSession, error)

	// Get the current running session of a client device.
	RunningSession(clnt IClientDevice) (cs IClientSession, ok bool)

	// Returns unconsumed session (if any) for the client device.
	AvailableSession(ctx context.Context, clnt IClientDevice) (IClientSession, error)

	// SessionSummary returns the session summary for the client device.
	SessionSummary(ctx context.Context, clnt IClientDevice) (*ClientSessionSummary, error)

	// FindSessionByID finds a session by its database ID and wraps it into an IClientSession object.
	// This is useful for displaying session information in templates and controllers
	// where you have a session ID from database queries but need access to SDK methods
	// like RemainingTime() and RemainingData() which account for elapsed time.
	FindSessionByID(ctx context.Context, sessionID int64) (IClientSession, error)

	// NewClientSession wraps session data into an IClientSession object without performing
	// additional database queries. This is useful when you already have session data from queries
	// and want to use SDK methods like RemainingTime() and RemainingData() which account for
	// elapsed time. The params parameter contains all session fields from the database row.
	NewClientSession(params NewClientSessionParams) IClientSession

	// NewClientDevice wraps device data into an IClientDevice object without performing
	// additional database queries. This is useful when you already have device data from queries
	// and want to use SDK methods like Update(), Emit(), and Subscribe(). The params parameter
	// contains all device fields from the database row.
	//
	// Deprecated: use ClientsMgr().NewClientDevice() instead. This method is kept here
	// for backward compatibility and delegates to the same underlying implementation.
	NewClientDevice(params NewDeviceParams) IClientDevice

	// NOTE: There is no session-list method on this API. To list/search/paginate
	// sessions, query the core `sessions` table directly with your plugin's own
	// sqlc queries (see the Core Database Tables guide), then wrap each row with
	// NewClientSession to get live RemainingTime()/RemainingData() calculations.

	// DeleteSession deletes a session by ID. If the session is currently running,
	// it disconnects the device first. Emits EventSessionBeforeDelete first (a callback
	// error cancels the deletion) and EventSessionDeleted after deletion.
	DeleteSession(ctx context.Context, sessionID int64) error

	// DeleteSessions deletes a batch of sessions by ID. It emits EventSessionBatchBeforeDelete
	// once before any deletion (subscribe via OnSessionBatchEvent; a callback error cancels
	// the whole batch), then deletes each session, emitting the per-session EventSessionDeleted.
	// The single-session EventSessionBeforeDelete is not fired per item.
	DeleteSessions(ctx context.Context, sessionIDs []int64) error

	// ListRunningSessions returns all currently active (running) sessions.
	// These are sessions that are actively connected and consuming time/data.
	// The returned sessions have real-time consumption data (RemainingTime/RemainingData
	// account for elapsed time since the session started).
	ListRunningSessions() ([]IClientSession, error)

	// FindRunningSessionByUUID finds a currently running session by its UUID.
	// Returns the session and true if found, or nil and false if no running session
	// exists with the given UUID. Unlike FindSessionByUUID which queries the database,
	// this method only checks in-memory running sessions for better performance when
	// you only need to know if a session is actively connected.
	FindRunningSessionByUUID(uuid string) (IClientSession, bool)

	// UpdateSession atomically applies the given field updates to a session and
	// persists them to the database in a single operation. If the session is
	// currently running, the update is routed to the live in-memory session
	// instance (the authoritative copy), so unsaved runtime consumption is never
	// lost and it is safe to pass a session object fetched earlier (e.g. via
	// FindSessionByID) even if it has become stale.
	//
	// Only non-nil fields in data are updated (same semantics as SetData).
	// Side effects for running sessions run after a successful persist: timer
	// reset (time fields), consumed-check (data fields), and TC rule updates
	// (bandwidth fields). EventSessionChanged is emitted unless
	// opts.IgnoreCallbacks is set. On a database error nothing is applied.
	//
	// Note: if the session was running, the update lands on the live instance;
	// the object you passed may be a stale snapshot afterward. Re-fetch via
	// RunningSession()/FindRunningSessionByUUID() if you need current values.
	//
	// This is the preferred way to modify a session, replacing the non-atomic
	// SetData() + Save() sequence.
	UpdateSession(ctx context.Context, session IClientSession, data SessionUpdateData, opts *SessionSaveOpts) error

	// MergeClientDevices merges the source device into the target device.
	// All sessions, purchases, and fingerprints are transferred from
	// source to target. The source device is deleted after the merge.
	//
	// Active sessions on either device are disconnected before the merge. If the
	// target device had an active session it is reconnected afterward.
	//
	// The OnClientMerge event is emitted after a successful merge so all registered
	// callbacks (e.g. cloud sync) are notified.
	//
	// Returns an error if the DB merge fails. Session disconnect/reconnect failures
	// are logged but do not abort the merge.
	//
	// Deprecated: use ClientsMgr().MergeClientDevices() instead. This method is kept here
	// for backward compatibility and delegates to the same underlying implementation.
	MergeClientDevices(ctx context.Context, targetID, sourceID int64) error
}
