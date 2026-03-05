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

// SessionEvent represents the type of a session event.
type SessionEvent string
type ClientEvent string
type PortalEvent string

const (
	EventSessionCreated      SessionEvent = "session:created"
	EventSessionConnected    SessionEvent = "session:connected"
	EventSessionDisconnected SessionEvent = "session:disconnected"
	EventSessionConsumed     SessionEvent = "session:expired"
	EventSessionChanged      SessionEvent = "session:changed"
	EventSessionDeleted      SessionEvent = "session:deleted"

	EventClientCreated      ClientEvent = "client:created"
	EventClientRegistered   ClientEvent = "client:registered"
	EventClientUpdated      ClientEvent = "client:updated"
	EventClientConnected    ClientEvent = "client:connected"
	EventClientDisconnected ClientEvent = "client:disconnected"
)

// SessionEventData represents the data associated with a session event.
type SessionEventData struct {
	Session       IClientSession
	ChangedFields SessionChangedFields // Which fields changed (only set for EventSessionChanged)
}

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
	ID         int64
	UUID       string
	MacAddress string
	IpAddress  string
	Hostname   string
	Status     DeviceStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// SessionFilterAvailability represents the availability filter for listing sessions.
type SessionFilterAvailability string

const (
	// SessionFilterAvailable returns sessions with remaining time/data that are not expired
	SessionFilterAvailable SessionFilterAvailability = "available"
	// SessionFilterConsumed returns sessions that are fully consumed (time/data exhausted)
	SessionFilterConsumed SessionFilterAvailability = "consumed"
	// SessionFilterExpired returns sessions that have passed their expiration date
	SessionFilterExpired SessionFilterAvailability = "expired"
)

type SessionPaymentType string

const (
	SessionVoucherPaymentType SessionPaymentType = "voucher"
	SessionCoinPaymentType    SessionPaymentType = "coin"
)

// ListSessionsParams holds parameters for listing sessions with pagination.
type ListSessionsParams struct {
	Search       *string                    // optional search by session UUID, device UUID/MAC/hostname/IP, provider package, or voucher code
	DeviceID     *int64                     // optional filter: sessions for a specific device ID
	Availability *SessionFilterAvailability // optional filter: "available", "consumed", or "expired"
	SessionType  *SessionType               // optional filter by session type: "time", "data", or "time-or-data"
	DateStart    *time.Time                 // optional filter: sessions created on or after this date (start of day)
	DateEnd      *time.Time                 // optional filter: sessions created on or before this date (end of day)
	TimeSecsGt   *int                       // optional filter: sessions with time_secs greater than this value
	TimeSecsLt   *int                       // optional filter: sessions with time_secs less than this value
	DataMbGt     *float64                   // optional filter: sessions with data_mbytes greater than this value
	DataMbLt     *float64                   // optional filter: sessions with data_mbytes less than this value
	Page         int
	PerPage      int
	PaymentType  *SessionPaymentType
}

// ListSessionsResult holds the result of listing sessions.
type ListSessionsResult struct {
	PaginatedSessions []IClientSession
	FilteredSessions  []IClientSession
}

// ISessionsMgrApi is used to manage client devices.
type ISessionsMgrApi interface {

	// Finds a client device by its ID.
	FindClientById(ctx context.Context, devId int64) (IClientDevice, error)

	// Finds a client by MAC address
	FindClientByMac(ctx context.Context, mac string) (IClientDevice, error)

	// FindClientByIp finds a client device by its IP address.
	// This is useful for scenarios where you have an IP address (e.g., from an HTTP request)
	// and need to find the associated device.
	FindClientByIp(ctx context.Context, ip string) (IClientDevice, error)

	// FindDeviceByUUID finds a client device by its globally unique identifier.
	// This is useful for referencing devices by their UUID rather than local database ID.
	FindDeviceByUUID(ctx context.Context, uuid string) (IClientDevice, error)

	// FindSessionByUUID finds a session by its globally unique identifier.
	// This is useful for querying or terminating sessions by their UUID.
	FindSessionByUUID(ctx context.Context, uuid string) (IClientSession, error)

	// Connects a client device to the internet.
	Connect(ctx context.Context, clnt IClientDevice, notify string) error

	// Disconnects a client device from the internet.
	// If notify is not nil, then the client device will be notified of the disconnection.
	Disconnect(ctx context.Context, clnt IClientDevice, notify string) error

	// Checks if a client device is connected to the internet.
	IsConnected(clnt IClientDevice) (connected bool)

	// Create a session for the client device
	CreateSession(ctx context.Context, params CreateSessionParams) (IClientSession, error)

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
	NewClientDevice(params NewDeviceParams) IClientDevice

	// OnSessionEvent registers a callback for session events.
	OnSessionEvent(event SessionEvent, callback func(data SessionEventData) error)

	// OnClientEvent registers a callback for client device events.
	OnClientEvent(event ClientEvent, callback func(clnt IClientDevice) error)

	// ListSessions returns a paginated list of sessions with optional search and filters.
	// Search matches against session UUID, device UUID/MAC/hostname/IP, provider package, or voucher code.
	// DeviceID filter returns only sessions for a specific device.
	// Availability filter: "all" (default), "available", "consumed", or "expired".
	// SessionType filter: "time", "data", or "time-or-data" (empty string means all types).
	// DateStart/DateEnd filter by session creation date (inclusive range).
	// TimeSecsGt/TimeSecsLt filter by allocated time in seconds.
	// DataMbGt/DataMbLt filter by allocated data in megabytes.
	ListSessions(ctx context.Context, params ListSessionsParams) (ListSessionsResult, error)

	// DeleteSession deletes a session by ID. If the session is currently running,
	// it disconnects the device first. Emits EventSessionDeleted after deletion.
	DeleteSession(ctx context.Context, sessionID int64) error

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
}
