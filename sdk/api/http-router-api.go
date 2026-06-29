/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "net/http"

type MuxRouteName string
type PluginRouteName string

type HttpRouterOpts struct {
	HttpsOnly bool // If true, the router will only serve HTTPS requests and redirect HTTP requests to HTTPS.
	Static    bool // If true, the router will serve static routes that persist across plugin version updates.
}

type AdminRouterOpts struct {
	Static bool // If true, the router will serve static routes that persist across plugin version updates.
}

type IHttpRouterApi interface {

	// AdminRouter returns a router whose routes require an authenticated admin
	// session. Admin routes are always served over HTTPS, so the only option is
	// Static (routes that persist across plugin version updates).
	AdminRouter(opts *AdminRouterOpts) IHttpRouterInstance

	// HttpRouter returns a general-purpose plugin router. Routes are accessible
	// over either scheme at /p/{package}/{version}/{path} with no authentication
	// middleware.
	HttpRouter(opts *HttpRouterOpts) IHttpRouterInstance

	// Register middlewares for captive portal page.
	// This middlewares are used to wrap the captive portal index page handler.
	UseForPortal(middlewares ...func(http.Handler) http.Handler)

	// Returns the url for the given route name.
	// If the route was registered on a static router, the static URL is returned automatically.
	UrlForRoute(name PluginRouteName, pairs ...string) (url string)

	// Returns the url for the route from third-party plugins.
	// This is used to create links to routes from other plugins.
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
