/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkhttp

import (
	"net/http"
)

type INavpsApi interface {
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
	NavCategorySystem   INavCategory = "system"
	NavCategoryPayments INavCategory = "payments"
	NavCategoryThemes   INavCategory = "themes"
	NavCategoryNetwork  INavCategory = "network"
	NavCategoryTools    INavCategory = "tools"
)

// AdminNavItemOpt represents an admin navigation menu item.
type AdminNavItemOpt struct {
	Category    INavCategory
	Label       string
	RouteName   string
	RouteParams map[string]string
}

type PortalNavItemOpt struct {
	Label       string
	IconUrl     string
	RouteName   string
	RouteParams map[string]string
}

type AdminNavList struct {
	Label string
	Items []AdminNavItem
}

type AdminNavItem struct {
	Label    string
	RouteUrl string
}

type PortalNavItem struct {
	Label    string
	IconUrl  string
	RouteUrl string
}
