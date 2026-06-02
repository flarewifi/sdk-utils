package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"core/utils/config"
	"core/utils/env"
)

// RedirectToPortalDomain funnels portal traffic to the shared captive-portal
// hostname over HTTPS (e.g. https://captive.flarewifi.com), so the portal is
// always served with its valid, cloud-issued certificate. Clients resolve that
// hostname to this router via split-horizon DNS (prod) or /etc/hosts (dev).
//
// It is a pass-through when the request is already on that hostname over HTTPS,
// or when no custom_domain is configured (preserving the legacy IP/HTTP flow).
func RedirectToPortalDomain() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg, err := config.GetCachedAppConfig()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			domain := strings.TrimSpace(cfg.CustomDomain)
			if domain == "" {
				next.ServeHTTP(w, r)
				return
			}

			host := r.Host
			if i := strings.IndexByte(host, ':'); i >= 0 {
				host = host[:i]
			}
			isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

			// Already on the portal hostname over HTTPS — serve normally.
			if strings.EqualFold(host, domain) && isHTTPS {
				next.ServeHTTP(w, r)
				return
			}

			http.Redirect(w, r, portalHTTPSURL(domain, r.URL.RequestURI()), http.StatusSeeOther)
		})
	}
}

// portalHTTPSURL builds the HTTPS portal URL, including the dev HTTPS port when
// it is not the standard 443.
func portalHTTPSURL(domain, uri string) string {
	if env.HTTPS_PORT == 443 {
		return fmt.Sprintf("https://%s%s", domain, uri)
	}
	return fmt.Sprintf("https://%s:%d%s", domain, env.HTTPS_PORT, uri)
}
