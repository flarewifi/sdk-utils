/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"net/http"
)

type INavsApi interface {
	// Register a factory function that returns the admin navigation menu items of your plugin.
	AdminNavsFactory(func(r *http.Request) []AdminNavItemOpt)

	// Register a factory function that returns the portal navigation menu items of your plugin.
	PortalNavsFactory(func(r *http.Request) []PortalNavItemOpt)

	// Returns the consolidated navigation list from all plugins for the admin dashboard.
	GetAdminNavs(r *http.Request) []AdminNavList

	// Returns the consolidated navigation list from all plugins for the portal.
	GetPortalItems(r *http.Request) []PortalNavItem
}

type INavCategory string

// List of admin navigation menu categories.
const (
	NavCategoryQuickAccess INavCategory = "quick_access"
	NavCategorySystem      INavCategory = "system"
	NavCategoryPayments    INavCategory = "payments"
	NavCategoryThemes      INavCategory = "themes"
	NavCategoryNetwork     INavCategory = "network"
	NavCategorySettings    INavCategory = "settings"
)

// AdminNavItemOpt represents an admin navigation menu item.
type AdminNavItemOpt struct {
	Category    INavCategory
	Label       string
	RouteName   string
	RouteParams map[string]string
	ExtraAttrs  map[string]any // HTML attributes for the menu item element (e.g., {"class": "custom-class", "data-id": "123"})
	Keywords    []string       // Used for admin nav search indexing
	Order       int            // Sort order within category (lower numbers appear first, default: 5000)
	Icon        string
}

type PortalNavItemOpt struct {
	Label       string
	IconFile    string
	RouteName   string
	RouteParams map[string]string
	ExtraAttrs  map[string]any // HTML attributes for the menu item element (e.g., {"class": "custom-class", "target": "_blank"})
	Metadata    any            // Custom metadata for theme plugins to use
}

type AdminNavList struct {
	Category INavCategory
	Label    string
	Items    []AdminNavItem
}

type AdminNavItem struct {
	Label      string
	RouteUrl   string
	IsCurrent  bool           // true if current active route
	Keywords   []string       // Used for admin nav search indexing
	ExtraAttrs map[string]any // HTML attributes for the menu item element (passed from AdminNavItemOpt)
	Order      int            // Sort order within category
	Icon       string
}

type PortalNavItem struct {
	ID         string
	Label      string
	IconUrl    string
	RouteUrl   string
	ExtraAttrs map[string]any // HTML attributes for the menu item element (passed from PortalNavItemOpt)
}
