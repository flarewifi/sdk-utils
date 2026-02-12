package api

import (
	"net/http"
	"net/url"
	sdkapi "sdk/api"
	"sort"
	"strings"
)

func NewNavsApi(api *PluginApi) *HttpNavsApi {
	return &HttpNavsApi{
		api: api,
		adminNavsFn: func(r *http.Request) (navs []sdkapi.AdminNavItemOpt) {
			return
		},
		portalNavsFn: func(r *http.Request) (navs []sdkapi.PortalNavItemOpt) {
			return
		},
	}
}

type HttpNavsApi struct {
	api          *PluginApi
	adminNavsFn  func(r *http.Request) []sdkapi.AdminNavItemOpt
	portalNavsFn func(r *http.Request) []sdkapi.PortalNavItemOpt
}

func (self *HttpNavsApi) AdminNavsFactory(fn func(r *http.Request) []sdkapi.AdminNavItemOpt) {
	self.adminNavsFn = fn
}

func (self *HttpNavsApi) PortalNavsFactory(fn func(r *http.Request) []sdkapi.PortalNavItemOpt) {
	self.portalNavsFn = fn
}

// GetAdminNavs returns the consolidated navigation list from all plugins for the admin dashboard.
// Navigation items are automatically sorted by the Order field within each category.
// Items without an Order value (or Order = 0) default to 5000.
func (self *HttpNavsApi) GetAdminNavs(r *http.Request) []sdkapi.AdminNavList {
	categories := []sdkapi.INavCategory{
		sdkapi.NavCategoryQuickAccess,
		sdkapi.NavCategoryPayments,
		sdkapi.NavCategorySystem,
		sdkapi.NavCategoryThemes,
		sdkapi.NavCategoryNetwork,
		sdkapi.NavCategoryTools,
	}

	categoryLabels := map[sdkapi.INavCategory]string{
		sdkapi.NavCategoryQuickAccess: self.api.CoreAPI.Translate("label", "Quick Access"),
		sdkapi.NavCategorySystem:      self.api.CoreAPI.Translate("label", "System"),
		sdkapi.NavCategoryThemes:      self.api.CoreAPI.Translate("label", "Themes"),
		sdkapi.NavCategoryPayments:    self.api.CoreAPI.Translate("label", "Payments"),
		sdkapi.NavCategoryNetwork:     self.api.CoreAPI.Translate("label", "Network"),
		sdkapi.NavCategoryTools:       self.api.CoreAPI.Translate("label", "Tools"),
	}

	navs := []sdkapi.AdminNavList{}
	for _, category := range categories {
		navItems := []sdkapi.AdminNavItem{}

		for _, p := range self.api.PluginsMgrApi.All() {
			navapi := p.Http().Navs().(*HttpNavsApi)
			adminNavs := navapi.adminNavsFn(r)
			for _, nav := range adminNavs {
				if nav.Category == category {
					routePairs := []string{}
					for k, v := range nav.RouteParams {
						routePairs = append(routePairs, k, v)
					}

					// Check if current url
					var isCurrent bool
					routeURL := p.Http().Helpers().UrlForRoute(nav.RouteName, routePairs...)
					parsed, err := url.Parse(routeURL)
					if parsed != nil && err == nil {
						isCurrent = strings.HasPrefix(r.URL.Path, parsed.Path) && !strings.Contains(routeURL, "not found")
					}

					// Set default order if not specified
					order := nav.Order
					if order == 0 {
						order = 5000 // Default middle priority
					}

					navItems = append(navItems, sdkapi.AdminNavItem{
						Label:      nav.Label,
						RouteUrl:   routeURL,
						IsCurrent:  isCurrent,
						Keywords:   nav.Keywords,
						ExtraAttrs: nav.ExtraAttrs, // Pass through HTML attributes for theme plugins
						Order:      order,
					})
				}
			}
		}

		// Sort nav items by Order field (ascending - lower numbers appear first)
		sort.Slice(navItems, func(i, j int) bool {
			return navItems[i].Order < navItems[j].Order
		})

		// Skip categories with no items
		if len(navItems) == 0 {
			continue
		}

		navs = append(navs, sdkapi.AdminNavList{
			Category: category,
			Label:    categoryLabels[category],
			Items:    navItems,
		})
	}

	return navs
}

// GetPortalItems returns the consolidated navigation list from all plugins for the portal.
// ExtraAttrs from PortalNavItemOpt are passed through to allow theme customization.
func (self *HttpNavsApi) GetPortalItems(r *http.Request) []sdkapi.PortalNavItem {
	items := []sdkapi.PortalNavItem{}
	for _, p := range self.api.PluginsMgrApi.All() {
		navsapi := p.Http().Navs().(*HttpNavsApi)
		portalItems := navsapi.portalNavsFn(r)
		for _, item := range portalItems {
			routePairs := []string{}
			for k, v := range item.RouteParams {
				routePairs = append(routePairs, k, v)
			}

			iconURL := ""
			if item.IconFile != "" {
				iconURL = p.Http().Helpers().PublicPath(item.IconFile)
			}

			url := ""
			if item.RouteName != "" {
				url = p.Http().Helpers().UrlForRoute(item.RouteName, routePairs...)
			}

			items = append(items, sdkapi.PortalNavItem{
				Label:      item.Label,
				IconUrl:    iconURL,
				RouteUrl:   url,
				ExtraAttrs: item.ExtraAttrs, // Pass through HTML attributes for theme plugins
			})
		}
	}
	return items
}
