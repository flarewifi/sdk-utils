//go:build dev

package api

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

// dnsServersDev is the list of DNS servers queried in parallel for hostname resolution.
// All servers are queried simultaneously and all unique IPs are collected.
var dnsServersDev = []string{
	"127.0.0.1:53",       // Local resolver (dnsmasq) - may have cached results
	"1.1.1.1:53",         // Cloudflare Primary
	"1.0.0.1:53",         // Cloudflare Secondary
	"8.8.8.8:53",         // Google Primary
	"8.8.4.4:53",         // Google Secondary
	"9.9.9.9:53",         // Quad9 Primary
	"149.112.112.112:53", // Quad9 Secondary
	"208.67.222.222:53",  // OpenDNS Primary
	"208.67.220.220:53",  // OpenDNS Secondary
	"94.140.14.14:53",    // AdGuard DNS Primary
	"94.140.15.15:53",    // AdGuard DNS Secondary
}

// ResolveHostnameToIps resolves a hostname to ALL possible IP addresses by querying
// all configured DNS servers in parallel and returning the union of all unique results.
// In dev mode, local development hostnames are mocked to avoid DNS resolution errors.
// Falls back to stale cached data if all DNS queries fail.
func (self *FirewallApi) ResolveHostnameToIps(hostname string) ([]string, error) {
	// Mock local development hostnames
	if strings.HasSuffix(hostname, ".flare-local.com") {
		return []string{"127.0.0.1"}, nil
	}

	// Check cache first
	self.dnsCacheMutex.RLock()
	cached, exists := self.dnsResolveCache[hostname]
	self.dnsCacheMutex.RUnlock()

	if exists && time.Since(cached.timestamp) < dnsResolveCacheTTL {
		return cached.ips, nil
	}

	// Cache expired or missing - query all DNS servers in parallel
	type dnsResult struct {
		ips []string
		err error
	}

	results := make(chan dnsResult, len(dnsServersDev))
	var wg sync.WaitGroup

	for _, dnsServer := range dnsServersDev {
		wg.Add(1)
		go func(server string) {
			defer wg.Done()

			resolver := &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{Timeout: 5 * time.Second}
					return d.DialContext(ctx, network, server)
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			ipList, err := resolver.LookupIP(ctx, "ip", hostname)
			cancel()

			if err != nil {
				results <- dnsResult{err: err}
				return
			}

			var ips []string
			for _, ip := range ipList {
				ips = append(ips, normalizeIPDev(ip.String()))
			}
			results <- dnsResult{ips: ips}
		}(dnsServer)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and deduplicate all IPs from all successful queries
	ipSet := make(map[string]bool)
	var lastErr error

	for result := range results {
		if result.err != nil {
			lastErr = result.err
			continue
		}
		for _, ip := range result.ips {
			ipSet[ip] = true
		}
	}

	// Convert set to sorted slice
	var allIps []string
	for ip := range ipSet {
		allIps = append(allIps, ip)
	}
	sortIPsDev(allIps)

	// Success: update cache and return
	if len(allIps) > 0 {
		self.dnsCacheMutex.Lock()
		self.dnsResolveCache[hostname] = dnsResolveEntry{
			ips:       allIps,
			timestamp: time.Now(),
		}
		self.dnsCacheMutex.Unlock()

		return allIps, nil
	}

	// All queries failed - fall back to stale cache if within max age
	if exists && time.Since(cached.timestamp) < dnsResolveCacheStaleMax {
		return cached.ips, nil
	}

	return nil, fmt.Errorf("failed to resolve hostname %s: all %d DNS servers failed (last error: %v), no cached data available", hostname, len(dnsServersDev), lastErr)
}

// normalizeIPDev normalizes an IP string, converting IPv4-mapped IPv6 addresses
// (e.g. ::ffff:192.0.2.1) to plain IPv4 (192.0.2.1) for proper deduplication.
func normalizeIPDev(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ipStr
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return ipv4.String()
	}
	return ip.String()
}

// sortIPsDev sorts IP addresses in-place: IPv4 addresses first, then IPv6,
// with ascending byte-order comparison within each group.
func sortIPsDev(ips []string) {
	sort.Slice(ips, func(i, j int) bool {
		ipA := net.ParseIP(ips[i])
		ipB := net.ParseIP(ips[j])

		if ipA == nil || ipB == nil {
			return ips[i] < ips[j]
		}

		isV4A := ipA.To4() != nil
		isV4B := ipB.To4() != nil

		if isV4A != isV4B {
			return isV4A // IPv4 before IPv6
		}

		return string(ipA) < string(ipB)
	})
}
