package middlewares

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"

	"core/internal/modules/logger"
	"core/internal/network"
	"core/utils/env"
	"core/utils/hostfinder"
	"core/utils/plugins"
)

// Portal-traffic claim middlewares: plugin hooks run BEFORE any funnel/scheme
// decision, at both funnel entry points (ForceHTTPS and the NotFoundHandler's
// RedirectToPortalDomain). A claim is a standard middleware — it inspects the
// request (typically the client IP from r.RemoteAddr) and either writes the
// response itself, taking full ownership of the page navigation, or calls
// next to leave the request to the normal funnel/portal flow. This is the
// backing store for the SDK's IHttpRouterApi.ClaimPortalTraffic — it lives
// here (not in the api package) because the api package imports middlewares;
// the reverse import would cycle.
//
// Copy-on-write + atomic.Pointer instead of a mutex: readers (every page
// navigation) do a single lock-free Load, and writers publish a freshly built
// slice via CAS. Registration is rare (plugin Init — including runtime plugin
// installs via store/devkit, which is why writes can race live traffic and
// unsynchronized access would be a data race), so the CAS retry loop is
// effectively free.
//
// Each entry carries the registering plugin's package alongside its
// middleware so applyPortalClaims can skip a claim belonging to a plugin that
// is currently blocked/disabled/update-skipped/queued for uninstall (see
// plugins.IsInvalid) — otherwise a plugin withheld from every other route
// (middlewares.PluginValidityCheck) would still be able to claim live
// captive-portal traffic through this funnel hook.
type portalClaim struct {
	pkg string
	mw  func(http.Handler) http.Handler
}

var portalClaims atomic.Pointer[[]portalClaim]

// RegisterPortalClaim adds portal-traffic claim middlewares owned by pkg.
// Claims wrap the funnel decision in registration order (first registered =
// outermost) on every top-level page navigation. Called via the SDK's
// ClaimPortalTraffic, once per plugin Init.
func RegisterPortalClaim(pkg string, claims ...func(http.Handler) http.Handler) {
	for {
		old := portalClaims.Load()
		var cur []portalClaim
		if old != nil {
			cur = *old
		}
		next := make([]portalClaim, 0, len(cur)+len(claims))
		next = append(next, cur...)
		for _, c := range claims {
			next = append(next, portalClaim{pkg: pkg, mw: c})
		}
		if portalClaims.CompareAndSwap(old, &next) {
			return
		}
	}
}

// This file is the single source of truth for the captive-portal funnel: the
// decision that routes a page navigation to the portal domain or leaves it in
// place. Two middlewares call it — ForceHTTPS (every matched route) and
// RedirectToPortalDomain (the NotFoundHandler, which mux Use middlewares never
// reach). Keeping the rule and its portal helpers here means the two funnel
// points can never drift apart.

// routePortalTraffic applies the captive-portal routing decision. Callers invoke
// it ONLY for a build that has a portal domain (config.HasCustomDomain) and a
// request that is a non-device-local, non-subresource page navigation:
//
//   - Non-GET requests are never funneled: a 302 makes browsers replay the
//     request as a GET, which breaks any form POST (e.g. the login POST at
//     /p/<pkg>/<ver>/login → 405). OS captive-detection probes are always
//     GETs, so the funnel loses nothing by ignoring other methods.
//   - Unmanaged source (an unmanaged/non-captive LAN, or a non-LAN interface such
//     as tailscale0 / a VPN / a routed PPPoE subscriber) → served as-is; the
//     funnel never redirects it anywhere. This is NOT captive-portal traffic, so
//     core does not act on it — mirroring the nft port-80 DNAT, which is likewise
//     scoped to the captive interfaces only.
//   - Already on the portal domain over the portal scheme → served as-is (returns
//     false; the caller hands the request to next).
//   - Anything else (a managed client not yet on the portal host/scheme) → 302 to
//     the portal domain. 302 is the redirect most universally followed by OS
//     captive-detection agents.
//
// isManagedRequest is conservative: only a source IP inside a LAN subnet whose
// captive portal is ENABLED counts as portal traffic — a captive-disabled
// interface, a VPN, or a lookup miss are all served as-is, never funneled to the
// portal domain.
//
// Returns true when it has written a redirect (the caller must stop) and false
// when the request should be served by next.
func routePortalTraffic(w http.ResponseWriter, r *http.Request) (handled bool) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}

	if !isManagedRequest(r) {
		logUnmanagedRequest(r)
		return false
	}

	domain := portalDomain()
	if strings.EqualFold(hostWithoutPort(r.Host), domain) && IsHTTPS(r) == (portalScheme() == "https") {
		return false
	}

	http.Redirect(w, r, portalURL(domain, r.URL.RequestURI()), http.StatusFound)
	return true
}

// IsManagedClient reports whether the request originates from a captive-portal
// managed client — the same classification the funnel uses (see isManagedRequest).
// Exported for handlers that must gate captive-only actions themselves, e.g. the
// portal device-registration controllers, which reject clients from non-captive
// networks now that the funnel no longer bounces them away first.
func IsManagedClient(r *http.Request) bool {
	return isManagedRequest(r)
}

// isManagedRequest reports whether the request should be funneled to the captive
// portal. It is true only when the client's source IP falls inside the subnet of
// a LAN interface whose captive portal is enabled (enable_captive_portal, with
// the primary-bridge device captive by default). An unmanaged/non-captive LAN, a
// non-LAN interface (tailscale0/VPN), or an IP we can't place in any LAN all
// return false — those requests are served as-is, never funneled.
func isManagedRequest(r *http.Request) bool {
	ip := clientIP(r)
	if ip == "" {
		return false
	}
	return network.IsClientIPManaged(ip)
}

// applyPortalClaims wraps next with every registered claim middleware, first
// registered outermost. It resolves the registry lazily per call (a lock-free
// atomic Load) because plugins register claims during Init, after the funnel
// middlewares are constructed. Zero-cost when no claims are registered — next
// is returned unchanged, so machines without claiming plugins pay nothing per
// request.
func applyPortalClaims(next http.Handler) http.Handler {
	p := portalClaims.Load()
	if p == nil {
		return next
	}

	claims := *p
	for i := len(claims) - 1; i >= 0; i-- {
		if plugins.IsInvalid(claims[i].pkg) {
			continue
		}
		next = claims[i].mw(next)
	}
	return next
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
// machine — the pass-through itself leaves no other trace.
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
