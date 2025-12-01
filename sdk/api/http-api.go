/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"net/http"
)

// IHttpApi is used to process and respond to http requests.
type IHttpApi interface {

	// Returns the current client device from http request.
	GetClientDevice(r *http.Request) (clnt IClientDevice, err error)

	// Returns the auth API.
	Auth() IHttpAuth

	// Returns the cookie API
	Cookie() IHttpCookie

	// Returns helper methods for views and handlers.
	Helpers() IHttpHelpers

	// Returns the router API.
	Router() IHttpRouterApi

	// Returns the http response writer API.
	Response() IHttpResponse

	// Returns the http variables in your routes. For example, if your route path is "/some/path/{varname}",
	// then you can get the value of "varname" by calling GetMuxVars(r)["varname"].
	MuxVars(r *http.Request) map[string]string

	// Returns the navs API.
	Navs() INavsApi

	// Returns the http forms API
	Forms() IHttpFormsApi
}
