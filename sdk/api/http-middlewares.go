/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "net/http"

// IHttpMiddlewares contains http middlewares for admin authentication, mobile device details, etc.
type IHttpMiddlewares interface {
	// Authenticate the user as admin.
	AdminAuth() func(http.Handler) http.Handler

	// Adds a cache-control header to the response.
	CacheResponse(days int) func(http.Handler) http.Handler

	// Checks if the user has a pending purchase. If yes, redirects to payment options page.
	PendingPurchase() func(http.Handler) http.Handler

	// Tracks navigation visits for the Quick Access menu.
	TrackNav() func(http.Handler) http.Handler

	// Authenticates internal webhook requests using JWT tokens.
	// Verifies the JWT token signed with application secret and adds device/purchase info to context.
	WebhookAuth() func(http.Handler) http.Handler
}
