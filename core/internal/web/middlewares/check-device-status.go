package middlewares

import (
	"fmt"
	"net/http"

	sdkapi "sdk/api"
)

func CheckDeviceStatus(api sdkapi.IPluginApi) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clnt, err := api.Http().GetClientDevice(r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			fmt.Println("status..", clnt.Status())
			fmt.Println("check device middleware...")

			navs := []sdkapi.PortalNavItemOpt{}
			switch clnt.Status() {
			case sdkapi.Blocked:
				navs = []sdkapi.PortalNavItemOpt{
					{
						Label:     "Blocked",
						RouteName: "portal.index",
					},
				}
			case sdkapi.Disconnected:
				navs = []sdkapi.PortalNavItemOpt{
					{
						Label:     "Paused",
						RouteName: "portal.index",
					},
				}
			}

			coreNavs := api.Http().Navs()
			coreNavs.PortalNavsFactory(func(r *http.Request) []sdkapi.PortalNavItemOpt {
				return navs
			})

			next.ServeHTTP(w, r)
		})
	}
}
