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

const (
	EventSessionConnected    SessionEvent = "session:connected"
	EventSessionDisconnected SessionEvent = "session:disconnected"
	EventSessionExpired      SessionEvent = "session:expired"
	EventSessionUpdated      SessionEvent = "session:updated"

	EventClientCreated      ClientEvent = "client:created"
	EventClientUpdated      ClientEvent = "client:updated"
	EventClientConnected    ClientEvent = "client:connected"
	EventClientDisconnected ClientEvent = "client:disconnected"
)

// SessionEventData represents the data associated with a session event.
type SessionEventData struct {
	Session IClientSession
	Device  IClientDevice
}

// ClientSessionSummary represents a summary of a client's session.
type ClientSessionSummary struct {
	RemainingTimeSecs   int
	RemainingDataMbytes float64
}

// CreateSessionParams holds parameters for creating a new client session.
type CreateSessionParams struct {
	DevId       int64
	SessionType SessionType
	TimeSecs    int
	DataMbytes  float64
	ExpDays     *int
	DownMbits   int
	UpMbits     int
	UseGlobal   bool
}

// NewSessionParams holds session data fields for wrapping an existing session row
// into an IClientSession object without performing database queries.
type NewSessionParams struct {
	ID              int64
	UUID            string
	ProviderPkg     string
	DeviceID        int64
	SessionType     SessionType
	TimeSecs        int
	DataMbytes      float64
	ConsumptionSecs int
	ConsumptionMb   float64
	StartedAt       *time.Time
	ResumedAt       *time.Time
	ExpDays         *int
	DownMbits       int
	UpMbits         int
	UseGlobal       bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
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

// ISessionsMgrApi is used to manage client devices.
type ISessionsMgrApi interface {

	// Finds a client device by its ID.
	FindClientById(ctx context.Context, devId int64) (IClientDevice, error)

	// FindDeviceByUUID finds a client device by its globally unique identifier.
	// This is useful for cloud sync scenarios where the cloud server needs to
	// reference devices by their UUID rather than local database ID.
	FindDeviceByUUID(ctx context.Context, uuid string) (IClientDevice, error)

	// FindSessionByUUID finds a session by its globally unique identifier.
	// This is useful for cloud sync scenarios where the cloud server needs to
	// terminate or query sessions by their UUID.
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

	// NewSession wraps session data into an IClientSession object without performing
	// additional database queries. This is useful when you already have session data from queries
	// and want to use SDK methods like RemainingTime() and RemainingData() which account for
	// elapsed time. The params parameter contains all session fields from the database row.
	NewSession(params NewSessionParams) IClientSession

	// NewClientDevice wraps device data into an IClientDevice object without performing
	// additional database queries. This is useful when you already have device data from queries
	// and want to use SDK methods like Update(), Emit(), and Subscribe(). The params parameter
	// contains all device fields from the database row.
	NewClientDevice(params NewDeviceParams) IClientDevice

	// OnSessionEvent registers a callback for session events.
	OnSessionEvent(event SessionEvent, callback func(data SessionEventData))

	// OnClientEvent registers a callback for client device events.
	OnClientEvent(event ClientEvent, callback func(clnt IClientDevice))
}
