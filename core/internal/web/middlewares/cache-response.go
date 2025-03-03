//go:build !dev

package middlewares

import (
	"fmt"
	"net/http"
	"time"
)

func CacheResponse(days int) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "max-age="+fmt.Sprintf("%d", days*24*60*60))
			w.Header().Set("Expires", fmt.Sprintf("%s", time.Now().AddDate(0, 0, days).Format(http.TimeFormat)))
			next.ServeHTTP(w, r)
		})
	}
}
