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
	// CssLibBootstrap5 is the only supported CSS library. Bootstrap 5 is provided
	// globally by core (admin + portal); Bootstrap 3 was removed when the machine
	// dropped old-browser support.
	CssLibBootstrap5 CSSLib = "bootstrap5"
)

type IThemesApi interface {
	// NewAdminTheme registers a new admin theme. The admin theme controls the layout
	// and appearance of the admin dashboard (post-login pages).
	// Only one admin theme can be active at a time.
	NewAdminTheme(AdminThemeOpts)

	// NewPortalTheme registers a new portal theme. The portal theme controls the layout
	// and appearance of the captive portal (user-facing pages like login and home).
	// Only one portal theme can be active at a time.
	NewPortalTheme(PortalThemeOpts)

	// GetAdminTheme returns the plugin API for the currently configured admin theme.
	// Returns nil if no admin theme is configured or if the theme plugin is not found.
	GetAdminTheme() IPluginApi

	// GetPortalTheme returns the plugin API for the currently configured portal theme.
	// Returns nil if no portal theme is configured or if the theme plugin is not found.
	GetPortalTheme() IPluginApi

	// AdminPreviewImage returns the preview image filename registered by this plugin's admin theme.
	// The filename is relative to the plugin's "resources/assets/public" folder.
	// Returns an empty string if this plugin has not registered an admin theme or no preview image was provided.
	AdminPreviewImage() string

	// PortalPreviewImage returns the preview image filename registered by this plugin's portal theme.
	// The filename is relative to the plugin's "resources/assets/public" folder.
	// Returns an empty string if this plugin has not registered a portal theme or no preview image was provided.
	PortalPreviewImage() string
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
	PreviewImage     string // image located in in resources/assets/public folder of the plugin, used for previewing the theme in the admin dashboard
	LayoutBuilder    func(w http.ResponseWriter, r *http.Request, builder IThemeComponents)
	IndexPageFactory func(w http.ResponseWriter, r *http.Request) ViewPage
}

type LoginPageData struct {
	LoginError error
}

type PortalThemeOpts struct {
	JsFile           string
	CssFile          string
	CssLib           CSSLib
	PreviewImage     string // image located in in resources/assets/public folder of the plugin, used for previewing the theme in the admin dashboard
	LayoutBuilder    func(w http.ResponseWriter, r *http.Request, builder IThemeComponents)
	LoginPageFactory func(w http.ResponseWriter, r *http.Request, data LoginPageData) ViewPage
	IndexPageFactory func(w http.ResponseWriter, r *http.Request) ViewPage
}
