package plugins

import (
	"net/http"
	sdkhttp "sdk/api/http"
)

func NewNavsApi(api *PluginApi) *HttpNavsApi {
	return &HttpNavsApi{
		api: api,
		adminNavsFn: func(r *http.Request) (navs []sdkhttp.AdminNavItemOpt) {
			return
		},
		portalNavsFn: func(r *http.Request) (navs []sdkhttp.PortalNavItemOpt) {
			return
		},
	}
}

type HttpNavsApi struct {
	api          *PluginApi
	adminNavsFn  func(r *http.Request) []sdkhttp.AdminNavItemOpt
	portalNavsFn func(r *http.Request) []sdkhttp.PortalNavItemOpt
}

func (self *HttpNavsApi) AdminNavsFactory(fn func(r *http.Request) []sdkhttp.AdminNavItemOpt) {
	self.adminNavsFn = fn
}

func (self *HttpNavsApi) PortalNavsFactory(fn func(r *http.Request) []sdkhttp.PortalNavItemOpt) {
	self.portalNavsFn = fn
}

func (self *HttpNavsApi) GetAdminNavs(r *http.Request) []sdkhttp.AdminNavList {
	categories := []sdkhttp.INavCategory{
		sdkhttp.NavCategorySystem,
		sdkhttp.NavCategoryPayments,
		sdkhttp.NavCategoryNetwork,
		sdkhttp.NavCategoryThemes,
		sdkhttp.NavCategoryTools,
	}

	navs := []sdkhttp.AdminNavList{}
	for _, category := range categories {
		navItems := []sdkhttp.AdminNavItem{}

		for _, p := range self.api.PluginsMgrApi.All() {
			navapi := p.Http().Navs().(*HttpNavsApi)
			adminNavs := navapi.adminNavsFn(r)
			for _, nav := range adminNavs {
				if nav.Category == category {
					routePairs := []string{}
					for k, v := range nav.RouteParams {
						routePairs = append(routePairs, k, v)
					}
					navItems = append(navItems, sdkhttp.AdminNavItem{
						Label:    nav.Label,
						RouteUrl: p.Http().Helpers().UrlForRoute(nav.RouteName, routePairs...),
					})
				}
			}
		}

		navs = append(navs, sdkhttp.AdminNavList{
			Label: self.api.CoreAPI.Utl.Translate("label", string(category)),
			Items: navItems,
		})
	}

	return navs
}

func (self *HttpNavsApi) GetPortalItems(r *http.Request) []sdkhttp.PortalNavItem {
	items := []sdkhttp.PortalNavItem{}
	for _, p := range self.api.PluginsMgrApi.All() {
		navsapi := p.Http().Navs().(*HttpNavsApi)
		portalItems := navsapi.portalNavsFn(r)
		for _, item := range portalItems {
			routePairs := []string{}
			for k, v := range item.RouteParams {
				routePairs = append(routePairs, k, v)
			}
			items = append(items, sdkhttp.PortalNavItem{
				Label:    item.Label,
				RouteUrl: p.Http().Helpers().UrlForRoute(item.RouteName, routePairs...),
			})
		}
	}
	return items
}
