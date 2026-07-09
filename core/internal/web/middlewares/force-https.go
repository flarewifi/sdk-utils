package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"core/internal/web/helpers"
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

// ForceHTTPS is the global middleware that fixes the scheme + host of admin and
// captive-portal traffic — but ONLY when this build has a portal domain
// (env.PortalDomain, i.e. staging/prod; never dev/devkit). It runs on RootRouter,
// which backs BOTH the HTTP and HTTPS listeners. The target depends on the path:
//
//   - Admin/device-local pages (see isDeviceLocalPath) are forced to HTTPS on the
//     SAME host, so the admin dashboard stays reachable by raw IP with no domain
//     (a cert-name warning is expected there).
//   - Portal/captive traffic is funneled to the portal domain over the portal
//     scheme: HTTPS on staging/prod (the valid cloud-issued cert). See portalScheme.
//
// When this build has NO portal domain (config.HasCustomDomain is false — dev/
// devkit) the machine serves a self-signed cert, so forcing a scheme would only
// produce cert warnings: this middleware becomes a pass-through and plain HTTP is
// served as-is.
//
// Port 80 stays open and REDIRECTS rather than drops, so OS captive-detection
// probes are still intercepted; HTTPS can't be transparently intercepted (a
// foreign probe host can't be handed a valid cert on :443), which is why portal
// traffic lands on the portal domain rather than the probe's own host.
func ForceHTTPS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// The scheme-force + funnel decision, wrapped by plugin portal-traffic
		// claims below so a claim can take ownership of a navigation before any
		// of it runs.
		tail := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Admin pages (/admin*, /login): never funneled, for every client —
			// managed or not. With a portal domain they are forced onto HTTPS on
			// the request's own host; without one (dev/devkit, self-signed cert)
			// they are served as-is.
			if isDeviceLocalPath(r.URL.Path) {
				if !config.HasCustomDomain() || IsHTTPS(r) {
					next.ServeHTTP(w, r)
					return
				}
				http.Redirect(w, r, httpsURL(hostWithoutPort(r.Host), r.URL.RequestURI()), http.StatusFound)
				return
			}

			// No portal domain (dev/devkit) => self-signed cert, no cloud-issued
			// host to funnel to, so there is nothing to funnel toward OR away from:
			// serve as-is. In dev everything stays reachable by raw IP/HTTP, the
			// legacy flow local tests rely on.
			if !config.HasCustomDomain() {
				next.ServeHTTP(w, r)
				return
			}

			// Everything else gets the shared funnel decision (unmanaged source or
			// non-GET → served as-is, a client already on the portal → served,
			// managed GET navigations → funneled to the portal domain). This is the
			// SAME routePortalTraffic the NotFoundHandler's RedirectToPortalDomain
			// calls, so the two funnel points can never diverge.
			if routePortalTraffic(w, r) {
				return
			}
			next.ServeHTTP(w, r)
		})

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sub-resource requests (assets, EventSource/XHR, favicon, images) are
			// scheme-agnostic: serve them on the scheme they were requested so they
			// MATCH the embedding page. The admin/portal scheme split applies to
			// top-level page navigations only — cross-scheme-redirecting a sub-resource
			// of an admin (HTTPS) page to the portal scheme (HTTP) trips the browser's
			// mixed-content blocker (and a redirect/retry loop). See isSubresourceRequest.
			if isSubresourceRequest(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Internal health checks are allowed to stay on plain HTTP.
			if httpsExemptPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Plugin portal-traffic claims (ClaimPortalTraffic) wrap the whole
			// scheme-force + funnel decision: a claim middleware can take full
			// ownership of any page navigation (typically by client IP), admin-
			// looking paths included, or fall through to it. NOT gated on
			// HasCustomDomain, so claims behave identically on dev and prod.
			// Sub-resources never reach the claims (bailed out above), so a
			// claimed page's core-served CSS/JS still load normally.
			applyPortalClaims(tail).ServeHTTP(w, r)
		})
	}
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// isSubresourceRequest reports whether a request is a sub-resource load (asset,
// EventSource/XHR, favicon, image, …) rather than a top-level page navigation. The
// admin/portal scheme split (see ForceHTTPS / RedirectToPortalDomain) must NOT
// apply to sub-resources: they have to be served on the scheme of their embedding
// page, or the browser blocks them as mixed content. Detection:
//   - Static asset paths (by extension) are always sub-resources.
//   - Modern browsers tag sub-resources with Sec-Fetch-Mode != "navigate".
//
// Clients that send no Sec-Fetch-Mode (older browsers, OS captive-portal probes)
// are treated as navigations so those probes are still funneled to the portal.
func isSubresourceRequest(r *http.Request) bool {
	if helpers.IsAssetPath(r.URL.Path) {
		return true
	}
	if mode := r.Header.Get("Sec-Fetch-Mode"); mode != "" && mode != "navigate" {
		return true
	}
	return false
}

// isDeviceLocalPath reports whether a path must stay on the machine's own host
// over HTTPS rather than being funneled to the portal domain: the admin UI and
// the admin login page ONLY. This keeps admin reachable by IP and never
// downgraded to HTTP — for every client, managed or not (it is also what keeps
// routed clients like PPPoE subscribers out of /admin: :443 is closed to them).
//
// The login form-POST handler (/p/<pkg>/<ver>/login, a generic plugin route) is
// deliberately NOT matched here anymore: it is protected by routePortalTraffic's
// GET-only guard instead (a funneled POST would replay as GET → 405), and it
// arrives over HTTPS anyway because the login page it is submitted from is.
// Activation pages are likewise no longer pinned to the machine host — a managed
// client's GET is funneled to the portal domain (same server via split-horizon
// DNS, valid cert), and activation POSTs pass through untouched.
func isDeviceLocalPath(path string) bool {
	return path == "/login" || strings.HasPrefix(path, "/admin")
}

// IsHTTPS reports whether the request arrived over TLS (directly or via a
// terminating proxy that set X-Forwarded-Proto). Exported so handlers outside
// this package can make the same scheme decision ForceHTTPS/RequireHTTPS use.
func IsHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

// HTTPSURL returns the HTTPS URL for the request's own host and URI (port
// normalized to the configured HTTPS listener). Exported so handlers can
// redirect a page onto HTTPS using the same host/port rules as the middlewares.
func HTTPSURL(r *http.Request) string {
	return httpsURL(hostWithoutPort(r.Host), r.URL.RequestURI())
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
