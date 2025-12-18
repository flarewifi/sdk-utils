//go:build dev

package api

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// ResolveHostnameToIps resolves a hostname to a list of IP addresses
// In dev mode, it mocks local development hostnames to avoid DNS resolution errors
func (self *FirewallApi) ResolveHostnameToIps(hostname string) ([]string, error) {
	// Mock local development hostnames
	if strings.HasSuffix(hostname, ".flare-local.com") {
		// Return localhost IP for local development domains
		return []string{"127.0.0.1"}, nil
	}

	// For non-local hostnames, use custom resolver with Cloudflare and Google DNS
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Second * 5,
			}
			// Try Cloudflare DNS first
			conn, err := d.DialContext(ctx, network, "1.1.1.1:53")
			if err != nil {
				// Fallback to Google DNS
				return d.DialContext(ctx, network, "8.8.8.8:53")
			}
			return conn, nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ipList, err := resolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve hostname %s: %v", hostname, err)
	}

	var ips []string
	for _, ip := range ipList {
		ips = append(ips, ip.String())
	}

	return ips, nil
}
