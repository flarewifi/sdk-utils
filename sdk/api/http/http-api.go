/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkhttp

import (
	"net/http"

	sdkconnmgr "sdk/api/connmgr"
)

// IHttpApi is used to process and respond to http requests.
type IHttpApi interface {

	// Returns the auth API.
	Auth() IHttpAuth

	// Returns helper methods for views and handlers.
	Helpers() IHttpHelpers

	Forms() IHttpFormApi

	// Returns the built in http middlewares
	Middlewares() IHttpMiddlewares

	// Returns the router API.
	HttpRouter() IHttpRouterApi

	// Returns the http response writer API.
	HttpResponse() IHttpResponse

	// Returns the navs API.
	Navs() INavpsApi

	// Returns the current client device from http request.
	GetClientDevice(r *http.Request) (clnt sdkconnmgr.IClientDevice, err error)

	// Returns the http variables in your routes. For example, if your route path is "/some/path/{varname}",
	// then you can get the value of "varname" by calling GetMuxVars(r)["varname"].
	MuxVars(r *http.Request) map[string]string
}
