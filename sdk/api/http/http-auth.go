/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkhttp

import (
	"net/http"

	sdkacct "sdk/api/accounts"
)

type IHttpAuth interface {

	// Get the current admin user from the http request.
	CurrentAcct(r *http.Request) (sdkacct.IAccount, error)

	// Check if the user is authenticated
	IsAuthenticated(r *http.Request) bool

	// Authenticate the user and return the account
	Authenticate(username string, password string) (sdkacct.IAccount, error)

	// Sets the auth-token cookie in response header
	SignIn(w http.ResponseWriter, acct sdkacct.IAccount) error

	// Sets empty auth-token cooke response header
	SignOut(w http.ResponseWriter) error
}
