/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkhttp

import (
	"net/http"

	"github.com/a-h/templ"
)

const (
	FlashSuccess string = "success"
	FlashError   string = "error"
	FlashWarning string = "warning"
)

// Scripts and styles are the index filenames in assets manifest
type ViewAssets struct {
	JsFile  string
	CssFile string
}

type ViewPage struct {
	Assets      ViewAssets
	PageContent templ.Component
}

// IHttpResponse is used to respond to http requests.
type IHttpResponse interface {

	// Used to render views from /resources/views/portal directory from your plugin.
	// For example if you have a view in /resources/views/portal/payment/index.html,
	// then you can render it with PortalView(w, r, "payment/index.html", data).
	// It uses the layout.html from your plugin directory /resources/views/portal/layout.html
	PortalView(w http.ResponseWriter, r *http.Request, v ViewPage)

	// Used to render views from /resources/views/admin directory from your plugin.
	// For example if you have a view in /resources/views/admin/dashboard/index.html,
	// then you can render it with AdminView(w, r, "dashboard/index.html", data).
	// It uses the layout.html from your plugin directory /resources/views/admin/layout.html
	AdminView(w http.ResponseWriter, r *http.Request, v ViewPage)

	// Used to render single file views (without layout) from /resources/views directory from your plugin.
	// For example if you have a view in /resources/views/index.html,
	// then you can render it with View(w, r, "index.html", data).
	View(w http.ResponseWriter, r *http.Request, v ViewPage)

	// Used to render resource files  from the resources directory in your plugin.
	// For example if you have a view in /resources/views/js/index.tmpl.js,
	// then you can render it with File(w, r, "views/js/index.tmpl.js", data).
	File(w http.ResponseWriter, r *http.Request, f string, data any)

	// Used to send json response.
	Json(w http.ResponseWriter, r *http.Request, data any, status int)

	Redirect(w http.ResponseWriter, r *http.Request, routeName string, pairs ...string)

	// Used to send flash messages.
	// For example, if you want to send a success message, you can use Flash(w, r, "Payment successful", sdkhttp.FlashSuccess)
	// Note that this does not send the http response immediately.
	// You need to call Flash(w, r, msg, typ) and then call any view rendering function to send the html that includes the flash message.
	FlashMsg(w http.ResponseWriter, r *http.Request, msg string, typ string)

	Error(w http.ResponseWriter, r *http.Request, err error, status int)
}
