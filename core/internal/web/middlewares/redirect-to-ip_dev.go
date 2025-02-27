//go:build dev

package middlewares

import (
	"net/http"
	sdkapi "sdk/api"
)

func RedirectToLanIP(api sdkapi.IPluginApi) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
