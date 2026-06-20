package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"core/utils/config"
	"core/utils/env"
)

// httpsExemptPaths are reachable over plain HTTP as well as HTTPS. These are
// device/internal health checks that local scripts poll without TLS and must get
// a 200 (or the boot status), never a redirect to a self-signed HTTPS endpoint.
var httpsExemptPaths = map[string]bool{
	"/ok":          true,
	"/boot/status": true,
}

// devPortalDomain is the fixed portal hostname used in development. The dev app
// config carries no custom_domain, but the cloud still issues a valid cert for
// this hostname, so dev always funnels HTTPS through it.
const devPortalDomain = "captive.flare-local.com"

// ForceHTTPS is the global middleware that makes both the admin pages and the
// captive portal always run over HTTPS. It runs on RootRouter, which backs BOTH
// the HTTP (:80) and HTTPS (:443) listeners, and redirects every plain-HTTP
// request to its HTTPS equivalent. The redirect TARGET host depends on the path:
//
//   - Admin/device-local pages (see isDeviceLocalPath) upgrade to HTTPS on the
//     SAME host, so the admin dashboard stays reachable by raw IP with no domain
//     (a cert-name warning is expected there).
//   - Portal/captive traffic is funneled to the portal domain — dev:
//     captive.flare-local.com (fixed), prod: the configured custom_domain — which
//     carries the valid cloud-issued cert and resolves to the device via
//     split-horizon DNS / /etc/hosts.
//
// Port 80 stays open and REDIRECTS rather than drops, so OS captive-detection
// probes are still intercepted; HTTPS can't be transparently intercepted (a
// foreign probe host can't be handed a valid cert on :443), which is why portal
// traffic lands on the portal domain rather than the probe's own host.
func ForceHTTPS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Already HTTPS — direct TLS, or terminated by a proxy that set the
			// forwarded-proto header. Serve as-is.
			if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
				next.ServeHTTP(w, r)
				return
			}

			// Internal health checks are allowed to stay on plain HTTP.
			if httpsExemptPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// 302 is the redirect most universally followed by OS captive-detection
			// agents; the portal/admin flows that might POST are always reached from
			// an already-HTTPS page, so they never hit this branch over HTTP.
			http.Redirect(w, r, httpsURL(redirectHost(r), r.URL.RequestURI()), http.StatusFound)
		})
	}
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// redirectHost resolves the HTTPS redirect target host. Admin/device-local paths
// stay on the request's own host so the admin dashboard works over IP with no
// domain; everything else (portal + captive) goes to the portal domain — the
// fixed dev hostname in development, the configured custom_domain in production
// (falling back to the request host when prod has none configured).
func redirectHost(r *http.Request) string {
	if isDeviceLocalPath(r.URL.Path) {
		return hostWithoutPort(r.Host)
	}
	if env.GO_ENV == env.ENV_DEV {
		return devPortalDomain
	}
	if cfg, err := config.GetCachedAppConfig(); err == nil {
		if d := strings.TrimSpace(cfg.CustomDomain); d != "" {
			return d
		}
	}
	return hostWithoutPort(r.Host)
}

// isDeviceLocalPath reports whether a path must stay on the device's own host
// (LAN IP / localhost) rather than being funneled to the portal domain: the admin
// UI, admin login, and the activation page. This keeps admin reachable by IP only.
func isDeviceLocalPath(path string) bool {
	return strings.HasPrefix(path, "/admin") ||
		path == "/login" ||
		strings.HasPrefix(path, "/activation")
}

func hostWithoutPort(host string) string {
	if i := strings.IndexByte(host, ':'); i >= 0 {
		return host[:i]
	}
	return host
}

func httpsURL(host, uri string) string {
	if env.HTTPS_PORT == 443 {
		return fmt.Sprintf("https://%s%s", host, uri)
	}
	return fmt.Sprintf("https://%s:%d%s", host, env.HTTPS_PORT, uri)
}
