package adminctrl

import (
	"core/internal/api"
	"fmt"
	"net/http"
	sdkapi "sdk/api"
	"slices"
)

func SearchCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		navsAPI := g.CoreAPI.Http().Navs()

		navList := navsAPI.GetAdminNavs(r)

		params := r.URL.Query()
		searchTxt := params.Get("q")

		var searchResult []sdkapi.AdminNavItem
		for _, nav := range navList {
			for _, item := range nav.Items {
				if slices.Contains(item.Keywords, searchTxt) {
					searchResult = append(searchResult, item)
				}
			}
		}

		fmt.Println("result: ", searchResult)
		// TODO: UI for search result
	}
}
