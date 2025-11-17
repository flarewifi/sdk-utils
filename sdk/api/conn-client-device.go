/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"context"
	"database/sql"
)

type DeviceStatus int

// List of device statuses.
const (
	Connected DeviceStatus = iota + 1
	Disconnected
	Blocked
)

// UpdateDeviceParams holds parameters for updating a client device.
type UpdateDeviceParams struct {
	Mac      string
	Ip       string
	Hostname string
	Status   int
}

// IClientDevice represents a client device connected to the network.
type IClientDevice interface {

	// Returns the database id of the client device ID.
	Id() int64

	// Returns the hostname of the device.
	Hostname() string

	// Returns the IP address of the device.
	IpAddr() string

	// Returns the MAC address of the device.
	MacAddr() string

	// Returns the status device status.
	Status() DeviceStatus

	// Updates the client device.
	Update(tx *sql.Tx, ctx context.Context, params UpdateDeviceParams) error

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
