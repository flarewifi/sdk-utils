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

type ILayoutBuilder interface {
	Content() templ.Component
	Render(head, layout templ.Component)
}

type LoginPageData struct {
	LoginUrl    string
	UsernameErr error
	PasswordErr error
}

type AdminThemeData struct {
	Navs    []AdminNavList
	Builder ILayoutBuilder
}

type AdminThemeOpts struct {
	CssLib           CSSLib
	JsFile           string
	CssFile          string
	LayoutFactory    func(w http.ResponseWriter, r *http.Request, data AdminThemeData)
	IndexPageFactory func(w http.ResponseWriter, r *http.Request) ViewPage
}

type PortalPageData struct {
	Navs []PortalNavItem
}

type PortalThemeData struct {
	Builder ILayoutBuilder
}

type PortalThemeOpts struct {
	JsFile           string
	CssFile          string
	CssLib           CSSLib
	LayoutFactory    func(w http.ResponseWriter, r *http.Request, data PortalThemeData)
	LoginPageFactory func(w http.ResponseWriter, r *http.Request, data LoginPageData) ViewPage
	IndexPageFactory func(w http.ResponseWriter, r *http.Request, data PortalPageData) ViewPage
}
