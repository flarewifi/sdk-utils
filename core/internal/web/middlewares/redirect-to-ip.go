//go:build !dev

package middlewares

import (
	"core/internal/network"
	"net/http"
	sdkapi "sdk/api"
)

func RedirectToLanIP(api sdkapi.IPluginApi) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res := api.Http().Response()
			clnt, err := api.Http().GetClientDevice(r)
			if err != nil {
				res.FlashMsg(w, r, api.Translate("error", "Unable to get client device"), sdkapi.FlashMsgError)
				next.ServeHTTP(w, r)
				return
			}

			ip := clnt.IpAddr()
			lan, err := network.FindByIp(ip)
			if err != nil {
				res.FlashMsg(w, r, api.Translate("error", "Unable to find network interface"), sdkapi.FlashMsgError)
				next.ServeHTTP(w, r)
				return
			}

			lanIP, err := lan.GetInterface().IpV4Addr()
			if err != nil {
				res.FlashMsg(w, r, api.Translate("error", "Unable to get network address"), sdkapi.FlashMsgError)
				next.ServeHTTP(w, r)
				return
			}

			if lanIP.Addr != r.Host {
				http.Redirect(w, r, "http://"+lanIP.Addr+r.URL.Path, http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
