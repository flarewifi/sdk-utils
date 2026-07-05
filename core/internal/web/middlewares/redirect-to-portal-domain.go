package middlewares

import (
	"net/http"
)

// RedirectToPortalDomain funnels portal traffic to the shared captive-portal
// hostname (env.PortalDomain) over HTTPS — the valid, cloud-issued cert on
// staging/prod. Clients resolve that hostname to this router via split-horizon
// DNS. See portalScheme.
//
// It exists only to back the NotFoundHandler in server.go: gorilla/mux runs
// Use() middlewares on matched routes ONLY, so a 404 executes with no middleware
// chain and must funnel itself. Matched routes get the identical decision from
// ForceHTTPS. Both delegate to routePortalTraffic (portal-funnel.go), the single
// source of truth for the funnel — so an unmanaged source (tailscale0/VPN) is
// bounced to /admin here too, never to the portal.
//
// It is a pass-through when the request is a sub-resource, or when this build has
// no portal domain (dev/devkit), preserving the legacy IP/HTTP flow.
func RedirectToPortalDomain() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sub-resources (assets, EventSource/XHR, favicon) must stay on their
			// embedding page's scheme/host — never funnel them, or the browser blocks
			// them as mixed content. See isSubresourceRequest.
			if isSubresourceRequest(r) {
				next.ServeHTTP(w, r)
				return
			}

			// No portal domain (dev/devkit) => nothing to funnel to; serve as-is.
			if portalDomain() == "" {
				next.ServeHTTP(w, r)
				return
			}

			if routePortalTraffic(w, r) {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
