/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "net/http"

type IHttpCookie interface {
	// SetCookie sets the cookie value for a given cookie name
	SetCookie(w http.ResponseWriter, name string, value string)

	// GetCookie returns the cookie value defined by the cookie name
	GetCookie(r *http.Request, name string) (value string, err error)

	// DeleteCookie deletes the cookie value for a given cookie name
	DeleteCookie(w http.ResponseWriter, name string)
}
