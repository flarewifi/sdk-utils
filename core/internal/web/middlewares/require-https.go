package middlewares

import (
	"net/http"
)

// RequireHTTPS forces every request through this router instance onto HTTPS.
// Unlike ForceHTTPS (the global, custom-domain-gated scheme fixer), this is
// unconditional: it backs the plugin HttpsRouter, whose routes must NEVER be
// served over plain HTTP. A non-TLS request is redirected to the same host/path
// on the HTTPS listener; an already-TLS request passes straight through.
//
// Both the HTTP and HTTPS listeners share RootRouter, so without this guard a
// route registered under /p/https would answer on port 80 too — exactly what an
// HTTPS-only router must prevent.
func RequireHTTPS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
			if isHTTPS {
				next.ServeHTTP(w, r)
				return
			}
			// 302 is the redirect most universally followed by clients and OS
			// captive-detection agents (matches ForceHTTPS' choice).
			http.Redirect(w, r, httpsURL(hostWithoutPort(r.Host), r.URL.RequestURI()), http.StatusFound)
		})
	}
}
