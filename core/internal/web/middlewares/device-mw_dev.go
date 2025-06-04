//go:build dev

package middlewares

import (
	"context"
	"fmt"
	"net"
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
				hostIP, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				ip = hostIP
			}

			h, err := hostfinder.FindByIp(ip)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if mac != "" {
				h.MacAddr = mac
			}

			dbpool := dtb.SqlDB()
			clnt, err := clntMgr.Register(dbpool, r, h.MacAddr, h.IpAddr, h.Hostname)
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
