/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkhttp

import (
	"html/template"
	"net/http"
)

// IHttpHelpers are methods available in html templates as .Helpers.
// For example, to use the Translate() method in html templates, use <% .Helpers.Translate "label" "network_settings" %>.
type IHttpHelpers interface {

	// Returns the uri of a file defined in asset manifest
	AssetPath(path string) (uri string)

	// Returns the html for the ads view.
	AdsView() (html template.HTML)

	CsrfHtmlTag(r *http.Request) string

	// Translates a message into the current language settings from application config.
	// msgtype is the message type, e.g. "error", "success", "info", "warning".
	// For example, if the current language is "en", then the following code in your template:
	//  <% .Helpers.Translate "error" "some-key" %>
	// will look for the file "/resources/translations/en/error/some-key.txt" under the plugin root directory
	// and displays the text inside that file.
	Translate(msgtype string, msgk string, pairs ...interface{}) string

	// Returns the url for the route.
	UrlForRoute(name string, pairs ...string) (uri string)

	// Returns the url from other plugins.
	UrlForPkgRoute(pkg string, name string, pairs ...string) (uri string)
}
