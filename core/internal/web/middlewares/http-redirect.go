package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"core/tools/env"
)

// HTTPRedirect redirects requests from HTTPS to HTTP
func HTTPRedirect() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if request is HTTPS
			isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

			if isHTTPS {
				// Build HTTP URL
				host := r.Host
				if strings.Contains(host, ":") {
					// Remove port if present
					host = strings.Split(host, ":")[0]
				}

				var httpURL string
				if env.HTTP_PORT == 80 {
					httpURL = fmt.Sprintf("http://%s%s", host, r.URL.RequestURI())
				} else {
					httpURL = fmt.Sprintf("http://%s:%d%s", host, env.HTTP_PORT, r.URL.RequestURI())
				}

				http.Redirect(w, r, httpURL, http.StatusMovedPermanently)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
