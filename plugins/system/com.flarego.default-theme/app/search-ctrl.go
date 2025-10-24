package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	sdkapi "sdk/api"
	"slices"
	"strings"
)

func SearchCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		navsAPI := api.Http().Navs()

		navList := navsAPI.GetAdminNavs(r)

		params := r.URL.Query()
		searchTxt := strings.ToLower(params.Get("q"))
		fmt.Println("searchTxt: ", searchTxt)

		fmt.Println("navList: ", navList)

		var searchResult []sdkapi.AdminNavItem
		for _, nav := range navList {
			for _, item := range nav.Items {
				fmt.Println("item.Keywords: ", item.Keywords)
				if slices.Contains(item.Keywords, searchTxt) {
					searchResult = append(searchResult, item)
				}
			}
		}

		fmt.Println("searchResult: ", searchResult)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"result": searchResult,
		})
		// TODO: UI for search result
	}
}
