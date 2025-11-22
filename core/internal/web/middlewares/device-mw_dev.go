//go:build dev

package middlewares

import (
	"context"
	"fmt"
	"net/http"

	"core/db"
	"core/internal/connmgr"
	"core/internal/utils/hostfinder"
	sdkapi "sdk/api"
)

func DeviceMiddleware(dtb *db.Database, clntMgr *connmgr.ClientRegister) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			clntSym := r.Context().Value(sdkapi.ClientCtxKey)
			if clntSym != nil {
				next.ServeHTTP(w, r)
				return
			}

			// Skip device middleware for internal webhook requests
			if r.Header.Get("X-Purchase-Webhook") == "true" {
				next.ServeHTTP(w, r)
				return
			}

			var ip, mac string

			macval, _ := r.Cookie("mac")
			ipval, _ := r.Cookie("ip")

			if macval != nil && macval.Value != "" {
				mac = macval.Value
			}

			if ipval != nil && ipval.Value != "" {
				ip = ipval.Value
			}

			if ip == "" {
				ip = "10.0.0.2"
			}

			h, err := hostfinder.FindByIp(ip)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if mac != "" {
				h.MacAddr = mac
			}

			if ip != "" {
				h.IpAddr = ip
			}

			clnt, err := clntMgr.Register(dtb, r, h.MacAddr, h.IpAddr, h.Hostname)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			fmt.Println("DeviceMiddleware: ", clnt)

			ctx := context.WithValue(r.Context(), sdkapi.ClientCtxKey, clnt)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
