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

type CSSLib string

const (
	CssLibBootstrap5 CSSLib = "bootstrap5"
	CssLibBootstrap3 CSSLib = "bootstrap3"
)

type IThemesApi interface {
	NewAdminTheme(AdminThemeOpts)
	NewPortalTheme(PortalThemeOpts)
}

type FlashMsg struct {
	Type    string
	Message string
}

type IThemeComponents interface {
	HtmlAttrs() templ.Attributes
	Head() templ.Component
	BodyAttrs() templ.Attributes
	PageContent() templ.Component
	Scripts() templ.Component
}

type AdminThemeOpts struct {
	CssLib           CSSLib
	JsFile           string
	CssFile          string
	LayoutBuilder    func(w http.ResponseWriter, r *http.Request, builder IThemeComponents)
	IndexPageFactory func(w http.ResponseWriter, r *http.Request) ViewPage
}

type LoginPageData struct {
	LoginUrl    string
	UsernameErr error
	PasswordErr error
}

type PortalThemeOpts struct {
	JsFile           string
	CssFile          string
	CssLib           CSSLib
	LayoutBuilder    func(w http.ResponseWriter, r *http.Request, builder IThemeComponents)
	LoginPageFactory func(w http.ResponseWriter, r *http.Request, data LoginPageData) ViewPage
	IndexPageFactory func(w http.ResponseWriter, r *http.Request) ViewPage
}
