/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"net/http"
)

type IHttpAuth interface {

	// Get the current admin user from the http request.
	// This will return error if AminAuth() middleware is not used.
	CurrentAcct(r *http.Request) (IAccount, error)

	// Check if the user is authenticated
	// This will perform cookie checks, also used in AdminAuth() middleware
	IsAuthenticated(r *http.Request) (IAccount, error)

	// Authenticate the user and return the account
	Authenticate(username string, password string) (IAccount, error)

	// Sets the auth-token cookie in response header
	SignIn(w http.ResponseWriter, acct IAccount) error

	// Sets an empty auth-token cookie response header
	SignOut(w http.ResponseWriter) error
}
