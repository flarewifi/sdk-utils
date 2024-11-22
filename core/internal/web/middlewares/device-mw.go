package middlewares

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"core/internal/connmgr"
	"core/internal/db"
	"core/internal/utils/hostfinder"
	sdkhttp "sdk/api/http"
)

func DeviceMiddleware(dtb *db.Database, clntMgr *connmgr.ClientRegister) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clntSym := r.Context().Value(sdkhttp.ClientCtxKey)
			if clntSym != nil {
				next.ServeHTTP(w, r)
				return
			}

			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			h, err := hostfinder.FindByIp(ip)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			clnt, err := clntMgr.Register(r, h.MacAddr, h.IpAddr, h.Hostname)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			fmt.Println("DeviceMiddleware: ", clnt)

			ctx := context.WithValue(r.Context(), sdkhttp.ClientCtxKey, clnt)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
