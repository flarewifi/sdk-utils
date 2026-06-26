package middlewares

import (
	"net/http"
	"strings"
)

// RedirectToPortalDomain funnels portal traffic to the shared captive-portal
// hostname (custom_domain) over the portal scheme — HTTPS in prod/staging (the
// valid, cloud-issued cert), plain HTTP in local dev (no cert for the dev portal
// host); see portalScheme. Clients resolve that hostname to this router via
// split-horizon DNS (prod) or /etc/hosts (dev).
//
// It is a pass-through when the request is already on that hostname over the
// portal scheme, or when no custom_domain is configured (preserving the legacy
// IP/HTTP flow).
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

			domain := portalDomain()
			if domain == "" {
				next.ServeHTTP(w, r)
				return
			}

			host := hostWithoutPort(r.Host)
			isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

			// Already on the portal hostname over the portal scheme — serve normally.
			if strings.EqualFold(host, domain) && isHTTPS == (portalScheme() == "https") {
				next.ServeHTTP(w, r)
				return
			}

			http.Redirect(w, r, portalURL(domain, r.URL.RequestURI()), http.StatusSeeOther)
		})
	}
}
