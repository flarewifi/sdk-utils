/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "net/http"

// IHttpMiddlewares contains http middlewares for admin authentication, mobile device details, etc.
type IHttpMiddlewares interface {
	// Returns the middleware for admin authentication.
	AdminAuth() func(next http.Handler) http.Handler

	// Returns the middleware for caching the response.
	// It forces browsers to cache the response for n number of days.
	CacheResponse(days int) func(next http.Handler) http.Handler

	// Returns the middleware that checks pending purchases
	// It rediercts to the pending purchase page if there is a pending purchase.
	PendingPurchase() func(next http.Handler) http.Handler
}
