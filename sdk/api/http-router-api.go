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

	// Register middlewares for captive portal page.
	// This middlewares are used to wrap the captive portal index page handler.
	UseForPortal(middlewares ...func(http.Handler) http.Handler)

	// Returns the url for the given route name.
	UrlForRoute(name PluginRouteName, pairs ...string) (url string)

	// Returns the url for the route from third-party plugins.
	// This is used create links to routes from other plugins.
	UrlForPkgRoute(pkg string, name string, pairs ...string) (url string)
}

type IHttpRouterInstance interface {

	// Register a subrouter for a given path
	Group(pattern string, fn func(subrouter IHttpRouterInstance))

	// Register a handler for a GET request to the given pattern.
	Get(pattern string, handler http.HandlerFunc, middlewares ...func(next http.Handler) http.Handler) (route IHttpRoute)

	// Register a handler for a POST request to the given pattern.
	Post(pattern string, handler http.HandlerFunc, middlewares ...func(next http.Handler) http.Handler) (route IHttpRoute)

	// Register a middleware to be used on all routes in this router instance.
	Use(middlewares ...func(next http.Handler) http.Handler)
}

// IHttpRoute represents a single route in the router.
type IHttpRoute interface {
	// Add url query params to the route
	Queries(pairs ...string) IHttpRoute

	// Set the name of the route.
	Name(name PluginRouteName) IHttpRoute
}
