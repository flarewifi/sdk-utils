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

type DeviceStatus int

// List of device statuses.
const (
	DeviceStatusConnected DeviceStatus = iota + 1
	DeviceStatusDisconnected
	DeviceStatusBlocked
)

// UpdateDeviceParams holds parameters for updating a client device.
type UpdateDeviceParams struct {
	UUID     string
	Mac      string
	Ip       string
	Hostname string
	Status   DeviceStatus
}

// DeviceData holds all device fields returned by Data() method.
// This struct is returned as a snapshot to minimize mutex usage.
type DeviceData struct {
	// Raw database values
	ID        int64
	UUID      string
	MacAddr   string
	IpAddr    string
	Hostname  string
	Status    DeviceStatus
	CreatedAt time.Time
	UpdatedAt time.Time

	// Pre-computed values
	IsConnected bool // True if device has an active internet session
}

// IClientDevice represents a client device connected to the network.
type IClientDevice interface {

	// Returns the database ID of the client device.
	ID() int64

	// Returns the UUID of the device.
	UUID() string

	// Returns the hostname of the device.
	Hostname() string

	// Returns the IP address of the device.
	IpAddr() string

	// Returns the MAC address of the device.
	MacAddr() string

	// Returns the device status.
	Status() DeviceStatus

	// Returns the creation timestamp of the device.
	CreatedAt() time.Time

	// Returns the last update timestamp of the device.
	UpdatedAt() time.Time

	// Returns a snapshot of all device data fields.
	// This method acquires the mutex once and returns all fields,
	// reducing lock contention compared to calling individual getters.
	Data() DeviceData

	// Updates the client device.
	Update(ctx context.Context, params UpdateDeviceParams) error

	// Emits a socket event to a client device.
	// The event will be propagated to the client's browser via server-sent events.
	Emit(event string, data []byte)

	// Subscribes to events for this client device.
	// It returns a channel that will receive data when the event is emitted.
	Subscribe(event string) <-chan []byte

	// Unsubscribes from events for this client device.
	// The channel argument comes from the Subscribe method.
	Unsubscribe(event string, ch <-chan []byte)
}
