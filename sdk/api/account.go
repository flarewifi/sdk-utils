/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

// IAccount represents a system account.
type IAccount interface {

	// Username returns the username for this account.
	Username() string

	// Get the permissions for this account.
	Permissions() []string

	// Update this account.
	Update(username string, password string, permissions []string) error

	// Delete this account.
	Delete() error

	// IsMaster checks if this account is a master account.
	IsMaster() bool

	// Check if account has all of the specified permissions.
	HasAllPerms(perms ...string) bool

	// Check if account has any of the specified permissions.
	HasAnyPerm(perms ...string) bool

	// Emit events to the browser for this account.
	// Events will be propagated to the client's browser via server-sent events.
	Emit(event string, data []byte)

	// Subscribe to events for this account.
	Subscribe(event string) <-chan []byte

	// Unsubscribe from events for this account.
	Unsubscribe(event string, ch <-chan []byte)
}
