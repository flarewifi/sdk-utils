/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkconnmgr

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// IClientDevice represents a client device connected to the network.
type IClientDevice interface {

	// Returns the database id of the client device ID.
	Id() pgtype.UUID

	// Returns the hostname of the device.
	Hostname() string

	// Returns the IP address of the device.
	IpAddr() string

	// Returns the MAC address of the device.
	MacAddr() string

	// Updates the client device.
	Update(ctx context.Context, mac string, ip string, hostname string) error

	// Emits a socket event to a client device.
	// The event will be propagated to the client's browser via server-sent events.
	Emit(event string, data any)

	// Subscribes to events for this client device.
	// It returns a channel that will receive data when the event is emitted.
	// The data is a JSON encoded byte slice.
	Subscribe(event string) <-chan []byte

	// Unsubscribes from events for this client device.
	// The channel argument comes from the Subscribe method.
	Unsubscribe(event string, ch <-chan []byte)
}
