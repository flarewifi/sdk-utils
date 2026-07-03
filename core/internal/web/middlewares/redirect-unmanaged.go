package middlewares

import (
	"net"
	"net/http"

	"core/internal/network"
	"core/utils/hostfinder"
)

// RedirectUnmanagedToAdmin gates the portal / device-registration flow so it
// only ever runs for traffic that positively originates from a MANAGED LAN
// interface. Any other request — a client on an UNMANAGED LAN, or one arriving
// on a non-LAN interface such as tailscale0 / a VPN tunnel, or one whose source
// IP can't be placed in any LAN subnet — is sent to /admin instead of being
// funneled to the captive-portal domain.
//
// Managed traffic is the only traffic whose flow the app touches; everything
// else must reach the machine directly (admin access), never the portal.
func RedirectUnmanagedToAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isManagedRequest(r) {
				http.Redirect(w, r, "/admin", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isManagedRequest reports whether the request positively comes from a managed
// LAN interface. It is intentionally conservative: only a source IP that falls
// inside a managed LAN's subnet counts as managed. An unmanaged LAN, a non-LAN
// interface (tailscale0/VPN), or an IP we can't place all return false so the
// caller diverts them to /admin rather than the portal.
func isManagedRequest(r *http.Request) bool {
	ip := clientIP(r)
	if ip == "" {
		return false
	}
	managed, known := network.IsClientIPManaged(ip)
	return known && managed
}

// clientIP resolves the client's source IP the same way the portal does: via
// hostfinder (the dev build reads the ip_addr cookie; prod resolves DHCP/ARP),
// falling back to the raw TCP peer address when hostfinder can't identify the
// host. The fallback matters for routed interfaces like tailscale0 that have no
// DHCP lease or ARP entry — there the TCP peer address is the only signal, and
// it still won't match any managed LAN subnet, so the request goes to /admin.
func clientIP(r *http.Request) string {
	if h, err := hostfinder.GetHostFromRequest(r); err == nil && h != nil && h.IpAddr != "" {
		return h.IpAddr
	}
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return ""
}
