/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"net/http"

	"github.com/a-h/templ"
)

const (
	FlashMsgSuccess string = "success"
	FlashMsgInfo    string = "info"
	FlashMsgWarning string = "warning"
	FlashMsgError   string = "error"
)

// Scripts and styles are the index filenames in assets manifest
type ViewAssets struct {
	JsFile  string
	CssFile string
}

type ViewPage struct {
	Assets        ViewAssets
	PageContent   templ.Component
	PreserveFlash bool // If true, flash cookies are not consumed (for intermediate redirect pages)
}

// IHttpResponse is used to respond to http requests.
type IHttpResponse interface {

	// Renders the page using the portal theme as the layout
	PortalView(w http.ResponseWriter, r *http.Request, v ViewPage)

	// Renders the page using the admin theme as the layout
	AdminView(w http.ResponseWriter, r *http.Request, v ViewPage)

	// Renderes the page without using any layout
	View(w http.ResponseWriter, r *http.Request, v ViewPage)

	// Used to send json response.
	Json(w http.ResponseWriter, r *http.Request, data any, status int)

	// Send HTTP redirect response to a given route name.
	Redirect(w http.ResponseWriter, r *http.Request, routeName string, pairs ...string)

	// Redirect to portal
	RedirectToPortal(w http.ResponseWriter, r *http.Request)

	// Redirect to a custom URL with a success message and 3-second delay
	// Shows "Redirecting..." message before redirecting to the specified URL
	RedirectSuccess(w http.ResponseWriter, r *http.Request, redirectURL string)

	// Used to send flash messages.
	// For example, if you want to send a success message, you can use Flash(w, r, "Payment successful", sdkapi.FlashMsgSuccess)
	// Note that this does not send the http response immediately.
	// You need to call Flash(w, r, msg, typ) and then call any view rendering function to send the html that includes the flash message.
	FlashMsg(w http.ResponseWriter, r *http.Request, msg string, typ string)

	// Used to show error messages
	Error(w http.ResponseWriter, r *http.Request, err error, status int)
}
