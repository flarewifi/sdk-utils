package middlewares

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"core/internal/modules/logger"
	"core/internal/network"
	"core/utils/env"
	"core/utils/hostfinder"
)

// This file is the single source of truth for the captive-portal funnel: the
// decision that routes a page navigation to the portal domain, to /admin, or
// leaves it in place. Two middlewares call it — ForceHTTPS (every matched route)
// and RedirectToPortalDomain (the NotFoundHandler, which mux Use middlewares
// never reach). Keeping the rule and its portal helpers here means the two funnel
// points can never drift apart.

// routePortalTraffic applies the captive-portal routing decision. Callers invoke
// it ONLY for a build that has a portal domain (config.HasCustomDomain) and a
// request that is a non-device-local, non-subresource page navigation:
//
//   - Unmanaged source (an unmanaged/non-captive LAN, or a non-LAN interface such
//     as tailscale0 / a VPN) → /admin on its own host over HTTPS. This is NOT
//     captive-portal traffic and must never reach the portal domain. It mirrors
//     the nft port-80 DNAT, which is likewise scoped to the captive interfaces
//     only — so an unmanaged interface is left untouched at BOTH the HTTP layer
//     (here) and the packet layer (nftables captive_ifaces set).
//   - Already on the portal domain over the portal scheme → served as-is (returns
//     false; the caller hands the request to next).
//   - Anything else (a managed client not yet on the portal host/scheme) → 302 to
//     the portal domain. 302 is the redirect most universally followed by OS
//     captive-detection agents.
//
// isManagedRequest is conservative: only a source IP inside a LAN subnet whose
// captive portal is ENABLED counts as portal traffic, so a captive-disabled
// interface, a VPN, or a lookup miss all fall through to /admin, never the portal.
//
// Returns true when it has written a redirect (the caller must stop) and false
// when the request is already on the portal and should be served by next.
func routePortalTraffic(w http.ResponseWriter, r *http.Request) (handled bool) {
	if !isManagedRequest(r) {
		logUnmanagedRequest(r)
		http.Redirect(w, r, httpsURL(hostWithoutPort(r.Host), "/admin"), http.StatusFound)
		return true
	}

	domain := portalDomain()
	if strings.EqualFold(hostWithoutPort(r.Host), domain) && IsHTTPS(r) == (portalScheme() == "https") {
		return false
	}

	http.Redirect(w, r, portalURL(domain, r.URL.RequestURI()), http.StatusFound)
	return true
}

// isManagedRequest reports whether the request should be funneled to the captive
// portal. It is true only when the client's source IP falls inside the subnet of
// a LAN interface whose captive portal is enabled (enable_captive_portal, with
// the primary-bridge device captive by default). An unmanaged/non-captive LAN, a
// non-LAN interface (tailscale0/VPN), or an IP we can't place in any LAN all
// return false.
func isManagedRequest(r *http.Request) bool {
	ip := clientIP(r)
	if ip == "" {
		return false
	}
	return network.IsClientIPManaged(ip)
}

// clientIP resolves the client's source IP the same way the portal does: via
// hostfinder (the dev build reads the ip_addr cookie; prod resolves DHCP/ARP),
// falling back to the raw TCP peer address when hostfinder can't identify the
// host. The fallback matters for routed interfaces like tailscale0 that have no
// DHCP lease or ARP entry — there the TCP peer address is the only signal, and
// it still won't match any managed LAN subnet.
func clientIP(r *http.Request) string {
	if h, err := hostfinder.GetHostFromRequest(r); err == nil && h != nil && h.IpAddr != "" {
		return h.IpAddr
	}
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return ""
}

// logUnmanagedRequest emits one diagnostic line for a request classified as NOT
// coming from a managed (captive) LAN: the request's host/path, the resolved
// client IP vs the raw TCP peer, and network.ClassifyClientIP's explanation of
// the verdict (which LAN matched, its cached device, or the full registry when
// nothing matched). This is what makes a misclassification debuggable on a real
// machine — the redirect itself is silent.
func logUnmanagedRequest(r *http.Request) {
	ip := clientIP(r)
	file, line := logger.GetCallerFileLine(1)
	logger.Emit(0, file, line, fmt.Sprintf(
		"Unmanaged request: host=%s path=%s clientIP=%s remoteAddr=%s — %s",
		r.Host, r.URL.Path, ip, r.RemoteAddr, network.ClassifyClientIP(ip),
	))
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
