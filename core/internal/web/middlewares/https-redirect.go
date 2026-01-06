//go:build !dev

package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"core/utils/env"
)

// HTTPSRedirect redirects admin routes from HTTP to HTTPS
func HTTPSRedirect() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if already HTTPS
			isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

			if !isHTTPS {
				// Build HTTPS URL
				host := r.Host
				if strings.Contains(host, ":") {
					// Remove port if present
					host = strings.Split(host, ":")[0]
				}

				var httpsURL string
				if env.HTTPS_PORT == 443 {
					httpsURL = fmt.Sprintf("https://%s%s", host, r.URL.RequestURI())
				} else {
					httpsURL = fmt.Sprintf("https://%s:%d%s", host, env.HTTPS_PORT, r.URL.RequestURI())
				}

				http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
