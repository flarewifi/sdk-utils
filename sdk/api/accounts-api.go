/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

const (
	AcctPermMaster = "master"
)

// IAccountsApi is used to manage accounts.
type IAccountsApi interface {

	// Create a new system account. The list of available permissions
	// can be obtained from IAcctApi.Permissions().
	Create(username string, pass string, perms []string) (IAccount, error)

	// Find an account by username.
	Find(username string) (IAccount, error)

	// Returns all normal and master admin accounts.
	GetAll() ([]IAccount, error)

	// Returns all master accounts.
	GetMasterAccts() ([]IAccount, error)

	// Add a new type of permission.
	NewPerm(name string, desc string) error

	// Returns all the permissions. It returns name and description (map) pairs of the permissions.
	GetAllPerms() map[string]string

	// Returns the permission description.
	PermDesc(perm string) (desc string)
}
