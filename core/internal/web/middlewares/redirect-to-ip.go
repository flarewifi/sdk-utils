//go:build !dev

package middlewares

import (
	"core/internal/network"
	"net"
	"net/http"
	sdkapi "sdk/api"
)

func RedirectToLanIP(api sdkapi.IPluginApi) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res := api.Http().Response()

			// Get IP from http request
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				res.FlashMsg(w, r, api.Translate("error", "Unable to parse remote address"), sdkapi.FlashMsgError)
				next.ServeHTTP(w, r)
				return
			}

			// Validate IP using net.ParseIP — works for both IPv4 and IPv6
			parsed := net.ParseIP(ip)
			if parsed == nil {
				res.FlashMsg(w, r, api.Translate("error", "Invalid client IP address"), sdkapi.FlashMsgError)
				next.ServeHTTP(w, r)
				return
			}
			isIPv6 := parsed.To4() == nil

			lan, err := network.FindByIp(ip)
			if err != nil {
				res.FlashMsg(w, r, api.Translate("error", "Unable to find network interface"), sdkapi.FlashMsgError)
				next.ServeHTTP(w, r)
				return
			}

			iface := lan.GetInterface()

			if isIPv6 {
				// For IPv6 clients redirect to the LAN's IPv6 address
				ipv6, err := iface.IpV6Addr()
				if err != nil {
					res.FlashMsg(w, r, api.Translate("error", "Unable to get network address"), sdkapi.FlashMsgError)
					next.ServeHTTP(w, r)
					return
				}
				// Wrap IPv6 address in brackets for the Host header
				lanHost := "[" + ipv6.Addr + "]"
				if lanHost != r.Host {
					http.Redirect(w, r, "http://"+lanHost+r.URL.Path, http.StatusSeeOther)
					return
				}
			} else {
				// For IPv4 clients redirect to the LAN's IPv4 address
				ipv4, err := iface.IpV4Addr()
				if err != nil {
					res.FlashMsg(w, r, api.Translate("error", "Unable to get network address"), sdkapi.FlashMsgError)
					next.ServeHTTP(w, r)
					return
				}
				if ipv4.Addr != r.Host {
					http.Redirect(w, r, "http://"+ipv4.Addr+r.URL.Path, http.StatusSeeOther)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
