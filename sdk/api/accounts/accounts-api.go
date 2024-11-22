/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkacct

// AccountsApi is used to manage accounts.
type AccountsApi interface {

	// Create a new system account. The list of available permissions
	// can be obtained from IAcctApi.Permissions().
	Create(username string, pass string, perms []string) (IAccount, error)

	// Find an account by username.
	Find(username string) (IAccount, error)

	// Get all accounts, admin and non-admin.
	GetAll() ([]IAccount, error)

	// Get all admin accounts.
	GetAdmins() ([]IAccount, error)

	// Add a new type of permission.
	NewPerm(name string, desc string) error

	// Retrieve all permissions.
	GetPerms() map[string]string

	// Retrieve a permission description.
	PermDesc(perm string) (desc string)
}
