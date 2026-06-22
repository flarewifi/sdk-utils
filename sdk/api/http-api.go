/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"net/http"
)

// RequestHost is the network identity of the client host behind an HTTP
// request, resolved from the machine's DHCP lease and ARP/NDP tables. Unlike
// IClientDevice it is NOT backed by a portal registration or device token — it
// is available for any LAN client that can reach the machine, including an admin
// browsing the dashboard who never went through the captive portal.
type RequestHost struct {
	// MacAddr is the uppercase MAC address, or "" if it could not be resolved.
	MacAddr string
	// IpAddr is the source IP of the request (IPv4 or IPv6).
	IpAddr string
	// Hostname is the DHCP-reported hostname, if any.
	Hostname string
}

// IHttpApi is used to process and respond to http requests.
type IHttpApi interface {

	// Returns the current client device from http request.
	GetClientDevice(r *http.Request) (clnt IClientDevice, err error)

	// GetRequestHost resolves the MAC/IP/hostname of the client host behind the
	// request from the machine's DHCP lease and ARP/NDP tables. Unlike
	// GetClientDevice it does NOT require a device token or a registered device
	// record, so it works for any LAN client (e.g. an admin on the dashboard).
	// Use it when you need the requesting device's MAC for a firewall change
	// regardless of portal state.
	GetRequestHost(r *http.Request) (host RequestHost, err error)

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
