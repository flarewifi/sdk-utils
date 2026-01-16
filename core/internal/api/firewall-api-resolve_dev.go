//go:build dev

package api

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// dnsServersDev is the ordered list of DNS servers to try for hostname resolution.
// Local resolver (dnsmasq) is tried first for cached results, then public DNS servers.
var dnsServersDev = []string{
	"127.0.0.1:53", // Local resolver (dnsmasq) - fastest, may have cached results
	"1.1.1.1:53",   // Cloudflare Primary
	"1.0.0.1:53",   // Cloudflare Secondary
	"8.8.8.8:53",   // Google Primary
	"8.8.4.4:53",   // Google Secondary
}

// ResolveHostnameToIps resolves a hostname to a list of IP addresses.
// In dev mode, it mocks local development hostnames to avoid DNS resolution errors.
// It tries multiple DNS servers sequentially until one succeeds.
func (self *FirewallApi) ResolveHostnameToIps(hostname string) ([]string, error) {
	// Mock local development hostnames
	if strings.HasSuffix(hostname, ".flare-local.com") {
		// Return localhost IP for local development domains
		return []string{"127.0.0.1"}, nil
	}

	var lastErr error

	for _, dnsServer := range dnsServersDev {
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: time.Second * 3}
				return d.DialContext(ctx, network, dnsServer)
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ipList, err := resolver.LookupIP(ctx, "ip", hostname)
		cancel()

		if err == nil && len(ipList) > 0 {
			var ips []string
			for _, ip := range ipList {
				ips = append(ips, ip.String())
			}
			return ips, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("failed to resolve hostname %s after trying all DNS servers: %v", hostname, lastErr)
}
