package api

import (
	"net/http"
	sdkapi "sdk/api"
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

func (self *HttpNavsApi) GetAdminNavs(r *http.Request) []sdkapi.AdminNavList {
	categories := []sdkapi.INavCategory{
		sdkapi.NavCategorySystem,
		sdkapi.NavCategoryPayments,
		sdkapi.NavCategoryNetwork,
		sdkapi.NavCategoryThemes,
		sdkapi.NavCategoryTools,
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
					navItems = append(navItems, sdkapi.AdminNavItem{
						Label:    nav.Label,
						RouteUrl: p.Http().Helpers().UrlForRoute(nav.RouteName, routePairs...),
					})
				}
			}
		}

		navs = append(navs, sdkapi.AdminNavList{
			Category: self.api.CoreAPI.Utl.Translate("label", string(category)),
			Items:    navItems,
		})
	}

	return navs
}

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
			items = append(items, sdkapi.PortalNavItem{
				Label:    item.Label,
				RouteUrl: p.Http().Helpers().UrlForRoute(item.RouteName, routePairs...),
			})
		}
	}
	return items
}
