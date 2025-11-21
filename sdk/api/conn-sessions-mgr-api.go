/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"context"
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

const (
	EventSessionConnected    SessionEvent = "session:connected"
	EventSessionDisconnected SessionEvent = "session:disconnected"
	EventSessionExpired      SessionEvent = "session:expired"
	EventSessionUpdated      SessionEvent = "session:updated"

	EventClientCreated      SessionEvent = "client:created"
	EventClientUpdated      SessionEvent = "client:updated"
	EventClientConnected    SessionEvent = "client:connected"
	EventClientDisconnected SessionEvent = "client:disconnected"
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

// ISessionsMgrApi is used to manage client devices.
type ISessionsMgrApi interface {

	// Finds a client device by its ID.
	FindClientById(ctx context.Context, devId int64) (IClientDevice, error)

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

	// OnSessionEvent registers a callback for session events.
	OnSessionEvent(event string, callback func(data SessionEventData))

	// OnClientEvent registers a callback for client device events.
	OnClientEvent(event string, callback func(clnt IClientDevice))
}
