/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkhttp

// IHttpRoute represents a single route in the router.
type IHttpRoute interface {
	// Add url query params to the route
	Queries(pairs ...string) IHttpRoute

	// Set the name of the route.
	Name(name PluginRouteName) IHttpRoute
}
