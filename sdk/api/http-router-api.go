/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "net/http"

type MuxRouteName string
type PluginRouteName string

type IHttpRouterApi interface {

	// Returns a plugin router with authentication middleware.
	AdminRouter() IHttpRouterInstance

	// Returns a generic plugin router.
	PluginRouter() IHttpRouterInstance

	// Register global middleware.
	Use(middlewares ...func(http.Handler) http.Handler)

	// Returns the url for the given route name.
	UrlForRoute(name PluginRouteName, pairs ...string) (url string)

	// Returns the url for the route from third-party plugins.
	// This is used create links to routes from other plugins.
	UrlForPkgRoute(pkg string, name string, pairs ...string) (url string)
}
