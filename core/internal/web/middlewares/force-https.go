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

			// No portal domain (dev/devkit) => self-signed cert, no cloud-issued
			// host to funnel to. Don't force any scheme; serve as-is.
			if !config.HasCustomDomain() {
				next.ServeHTTP(w, r)
				return
			}

			isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

			// Admin/device-local pages: always HTTPS on the request's own host.
			if isDeviceLocalPath(r.URL.Path) {
				if isHTTPS {
					next.ServeHTTP(w, r)
					return
				}
				http.Redirect(w, r, httpsURL(hostWithoutPort(r.Host), r.URL.RequestURI()), http.StatusFound)
				return
			}

			// A request arriving on an UNMANAGED interface (an unmanaged LAN, or a
			// non-LAN interface such as tailscale0 / a VPN) is not captive-portal
			// traffic: send it to the admin UI on its own host over HTTPS rather
			// than funneling it to the portal domain. This runs after the
			// device-local / asset / health-check guards above, so /admin itself is
			// already excluded and there is no redirect loop. isManagedRequest is
			// conservative: only a source IP inside a MANAGED LAN subnet is treated
			// as portal traffic, so a UBUS/lookup miss falls through to /admin, never
			// the portal.
			if !isManagedRequest(r) {
				http.Redirect(w, r, httpsURL(hostWithoutPort(r.Host), "/admin"), http.StatusFound)
				return
			}

			// Captive portal pages: funnel to the portal domain over the portal
			// scheme (HTTPS on staging/prod). Already there => serve.
			domain := portalDomain()
			if domain != "" && strings.EqualFold(hostWithoutPort(r.Host), domain) && isHTTPS == (portalScheme() == "https") {
				next.ServeHTTP(w, r)
				return
			}
			// 302 is the redirect most universally followed by OS captive-detection agents.
			http.Redirect(w, r, portalURL(domain, r.URL.RequestURI()), http.StatusFound)
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

// portalDomain returns this build's captive-portal hostname, or "" when there is
// none. It is derived purely from the build environment (empty in dev/devkit,
// captive.nexifi.ph on staging, captive.flarewifi.com on prod) — application.json's
// custom_domain is intentionally ignored for now. See env.PortalDomain.
func portalDomain() string {
	return env.PortalDomain()
}

// portalScheme is the scheme captive-portal pages are served over: plain HTTP in
// local dev (no valid cert for the dev portal host) and HTTPS everywhere else (the
// cloud-issued cert). Admin pages are always HTTPS regardless of this.
func portalScheme() string {
	if env.GO_ENV == env.ENV_DEV {
		return "http"
	}
	return "https"
}

// portalURL builds the captive-portal URL on the given host using portalScheme and
// the matching listener port (omitting the port when it is the scheme default).
func portalURL(host, uri string) string {
	scheme := portalScheme()
	port := env.HTTPS_PORT
	if scheme == "http" {
		port = env.HTTP_PORT
	}
	if (scheme == "https" && port == 443) || (scheme == "http" && port == 80) {
		return fmt.Sprintf("%s://%s%s", scheme, host, uri)
	}
	return fmt.Sprintf("%s://%s:%d%s", scheme, host, port, uri)
}

// isDeviceLocalPath reports whether a path must stay on the device's own host over
// HTTPS rather than being funneled to the portal domain (which is plain HTTP in dev):
// the admin UI, admin login, and the activation page. This keeps admin reachable by IP
// only and never downgraded to HTTP.
//
// The login is matched by suffix because its render path (/login) and its form-POST
// handler live on different prefixes — the handler is a generic plugin route
// (/p/<pkg>/<ver>/login), NOT an /admin route — yet both must stay on HTTPS.
func isDeviceLocalPath(path string) bool {
	return strings.HasPrefix(path, "/admin") ||
		strings.HasSuffix(path, "/login") ||
		strings.HasPrefix(path, "/activation")
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
