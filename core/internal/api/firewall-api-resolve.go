//go:build !dev

package api

import (
	"context"
	"fmt"
	"net"
	"time"
)

// ResolveHostnameToIps resolves a hostname to a list of IP addresses using Cloudflare and Google DNS
func (self *FirewallApi) ResolveHostnameToIps(hostname string) ([]string, error) {
	// Use custom resolver with Cloudflare (1.1.1.1) and Google (8.8.8.8) DNS servers
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
