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
	CssLibBootstrap5 AdminCssLib  = "bootstrap5"
	CssLibBootstrap3 PortalCssLib = "bootstrap3"
)

type AdminCssLib string
type PortalCssLib string

type IHttpThemesApi interface {
	NewAdminTheme(AdminThemeOpts)
	NewPortalTheme(PortalThemeOpts)
}

type FlashMsg struct {
	Type    string
	Message string
}

type PageAssets struct {
	GlobalJsSrc   string
	GlobalCssHref string
	ThemeJsSrc    string
	ThemeCssHref  string
	PageJsSrc     string
	PageCssHref   string
}

type LayoutData struct {
	Assets      PageAssets
	FlashMsg    FlashMsg
	PageContent templ.Component
}

type AdminLayoutData struct {
	Layout LayoutData
	Navs   []AdminNavList
}

type PortalLayoutData struct {
	Layout LayoutData
}

type LoginPageData struct {
	LoginUrl    string
	UsernameErr error
	PasswordErr error
}

type PortalIndexData struct {
	Navs []PortalNavItem
}

type AdminThemeOpts struct {
	JsFile           string
	CssFile          string
	CssLib           AdminCssLib
	LayoutFactory    func(w http.ResponseWriter, r *http.Request, data AdminLayoutData) templ.Component
	IndexPageFactory func(w http.ResponseWriter, r *http.Request) ViewPage
}

type PortalThemeOpts struct {
	JsFile           string
	CssFile          string
	CssLib           PortalCssLib
	LayoutFactory    func(w http.ResponseWriter, r *http.Request, data PortalLayoutData) templ.Component
	LoginPageFactory func(w http.ResponseWriter, r *http.Request, data LoginPageData) ViewPage
	IndexPageFactory func(w http.ResponseWriter, r *http.Request, data PortalIndexData) ViewPage
}
