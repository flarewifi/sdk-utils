/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkacct

// IAccount represents a system account.
type IAccount interface {

	// Username returns the username for this account.
	Username() string

	// IsAdmin checks if this account is an admin.
	IsAdmin() bool

	// Get the permissions for this account.
	Permissions() []string

	// Check if account has all of the specified permissions.
	HasAllPerms(perms ...string) bool

	// Check if account has any of the specified permissions.
	HasAnyPerm(perms ...string) bool

	// Update this account.
	Update(username string, password string, permissions []string) error

	// Delete this account.
	Delete() error

	// Emit events to the browser for this account.
	// Events will be propagated to the client's browser via server-sent events.
	Emit(event string, data interface{})

	// Subscribe to events for this account.
	Subscribe(event string) <-chan []byte

	// Unsubscribe from events for this account.
	Unsubscribe(event string, ch <-chan []byte)
}
