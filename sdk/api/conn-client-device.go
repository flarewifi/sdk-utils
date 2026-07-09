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
	Ipv4     string
	Ipv6     string
	Hostname string
	Status   DeviceStatus
}

// DeviceData holds all device fields returned by Data() method.
// This struct is returned as a snapshot to minimize mutex usage.
type DeviceData struct {
	// Raw database values
	ID          int64
	UUID        string
	CookieToken string
	MacAddr     string
	Ipv4Addr    string
	Ipv6Addr    string
	Hostname    string
	Status      DeviceStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Pre-computed values
	IsConnected bool // True if device has an active internet session
}

// IClientDevice represents a client device — a phone, tablet, laptop/PC, or other
// end-user host — connected to the machine's network. This is NOT the machine
// (the OpenWRT router/hotspot box) itself; see IMachineApi for that.
type IClientDevice interface {

	// Returns the database ID of the client device.
	ID() int64

	// Returns the UUID of the device.
	UUID() string

	// Returns the cookie token used for cookie validation.
	// An empty string means no cookie token validation is enforced.
	CookieToken() string

	// Returns the hostname of the device.
	Hostname() string

	// Returns the IPv4 address of the device (empty if not available).
	Ipv4Addr() string

	// Returns the IPv6 address of the device (empty if not available).
	Ipv6Addr() string

	// IpAddr returns the primary IP address for backward compatibility.
	// Returns IPv4 if available, otherwise IPv6.
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
