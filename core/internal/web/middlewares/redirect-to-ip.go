//go:build !dev

package middlewares

import (
	"core/internal/network"
	"net"
	"net/http"
	"regexp"
	sdkapi "sdk/api"
)

func RedirectToLanIP(api sdkapi.IPluginApi) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res := api.Http().Response()

			// Get ip from http request
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				res.FlashMsg(w, r, api.Translate("error", "Unable to parse remote address"), sdkapi.FlashMsgError)
				next.ServeHTTP(w, r)
				return
			}

			// IPv4 pattern check
			pattern := `^(\d{1,3}\.){3}\d{1,3}$`
			if matched, _ := regexp.Match(pattern, []byte(ip)); !matched {
				res.FlashMsg(w, r, api.Translate("error", "Invalid client IP address"), sdkapi.FlashMsgError)
				next.ServeHTTP(w, r)
				return
			}

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
