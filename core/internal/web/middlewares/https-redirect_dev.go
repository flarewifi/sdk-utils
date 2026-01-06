//go:build dev

package middlewares

import (
	"net/http"
)

// HTTPSRedirect is disabled in dev mode - allows HTTP traffic on port 3000
func HTTPSRedirect() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// In dev mode, allow HTTP traffic without redirecting
			next.ServeHTTP(w, r)
		})
	}
}
