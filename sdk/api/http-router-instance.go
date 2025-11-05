/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "net/http"

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
