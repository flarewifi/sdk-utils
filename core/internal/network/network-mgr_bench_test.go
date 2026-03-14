package network

import (
	"fmt"
	"net"
	"sync"
	"testing"
)

// ============================================================================
// BENCHMARK HELPERS
// ============================================================================

// benchLAN holds pre-parsed data representing what the current FindByIp()
// fetches dynamically on every call (addr + netmask from UBUS, CIDR parse).
type benchLAN struct {
	name       string
	lan        *NetworkLan
	ipv4Addr   string // e.g. "192.168.1.1"
	netmask    int    // e.g. 24
	cidrString string // e.g. "192.168.1.0/24"
	cidr       *net.IPNet
}

func makeBenchLANs(n int) []benchLAN {
	lans := make([]benchLAN, n)
	for i := 0; i < n; i++ {
		addr := fmt.Sprintf("192.168.%d.1", i+1)
		cidrStr := fmt.Sprintf("192.168.%d.1/24", i+1)
		_, cidr, _ := net.ParseCIDR(cidrStr)
		lans[i] = benchLAN{
			name:       fmt.Sprintf("lan%d", i),
			lan:        NewNetworkLan(fmt.Sprintf("lan%d", i)),
			ipv4Addr:   addr,
			netmask:    24,
			cidrString: cidrStr,
			cidr:       cidr,
		}
	}
	return lans
}

// ============================================================================
// OLD IMPLEMENTATION — inline replica of what FindByIp() does today
// ============================================================================

// findByIpOld mirrors the exact logic of the current FindByIp() but operates
// on a slice of benchLANs instead of querying UBUS, so we can measure the
// pure algorithmic cost: sync.Map.Range + per-iteration ParseCIDR + ParseIP.
func findByIpOld(lans []benchLAN, clientIp string) (*NetworkLan, error) {
	var result *NetworkLan

	// Simulate sync.Map.Range over the LAN entries
	m := &sync.Map{}
	for _, l := range lans {
		m.Store(l.name, l)
	}

	m.Range(func(key, value any) bool {
		l := value.(benchLAN)

		// Current code: builds cidrString and calls net.ParseCIDR on every call
		cidrStr := fmt.Sprintf("%s/%d", l.ipv4Addr, l.netmask)
		_, lanCidr, err := net.ParseCIDR(cidrStr)
		if err != nil {
			return true
		}

		// Current code: calls net.ParseIP on every iteration
		ip := net.ParseIP(clientIp)
		if ip == nil {
			return false
		}

		if lanCidr.Contains(ip) {
			result = l.lan
			return false
		}
		return true
	})

	if result == nil {
		return nil, fmt.Errorf("no matching LAN found for IP %s", clientIp)
	}
	return result, nil
}

// ============================================================================
// NEW IMPLEMENTATION — optimized: pre-parsed CIDRs, RWMutex, slice scan
// ============================================================================

type lanEntryBench struct {
	name string
	lan  *NetworkLan
	cidr *net.IPNet // pre-parsed once at addLan() time
}

type lanRegistryBench struct {
	mu    sync.RWMutex
	byIp  []*lanEntryBench
	count int
}

func newRegistryBench(lans []benchLAN) *lanRegistryBench {
	r := &lanRegistryBench{
		byIp: make([]*lanEntryBench, 0, len(lans)),
	}
	for _, l := range lans {
		r.byIp = append(r.byIp, &lanEntryBench{
			name: l.name,
			lan:  l.lan,
			cidr: l.cidr, // pre-parsed
		})
		r.count++
	}
	return r
}

func findByIpNew(r *lanRegistryBench, clientIp string) (*NetworkLan, error) {
	// Parse client IP once (outside lock)
	ip := net.ParseIP(clientIp)
	if ip == nil {
		return nil, fmt.Errorf("invalid client IP: %s", clientIp)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, entry := range r.byIp {
		if entry.cidr.Contains(ip) {
			return entry.lan, nil
		}
	}

	return nil, fmt.Errorf("no matching LAN found for IP %s", clientIp)
}

// ============================================================================
// BENCHMARKS — FindByIp: Old vs New
// ============================================================================

// BenchmarkFindByIp_Old_SingleLAN: 1 LAN, IP matches — typical hotspot setup
func BenchmarkFindByIp_Old_SingleLAN(b *testing.B) {
	lans := makeBenchLANs(1)
	clientIp := "192.168.1.100"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = findByIpOld(lans, clientIp)
	}
}

func BenchmarkFindByIp_New_SingleLAN(b *testing.B) {
	lans := makeBenchLANs(1)
	r := newRegistryBench(lans)
	clientIp := "192.168.1.100"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = findByIpNew(r, clientIp)
	}
}

// BenchmarkFindByIp_*_ThreeLANs_First: 3 LANs, IP matches first LAN
func BenchmarkFindByIp_Old_ThreeLANs_First(b *testing.B) {
	lans := makeBenchLANs(3)
	clientIp := "192.168.1.100" // matches lans[0]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = findByIpOld(lans, clientIp)
	}
}

func BenchmarkFindByIp_New_ThreeLANs_First(b *testing.B) {
	lans := makeBenchLANs(3)
	r := newRegistryBench(lans)
	clientIp := "192.168.1.100"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = findByIpNew(r, clientIp)
	}
}

// BenchmarkFindByIp_*_ThreeLANs_Last: 3 LANs, IP matches last LAN (worst case scan)
func BenchmarkFindByIp_Old_ThreeLANs_Last(b *testing.B) {
	lans := makeBenchLANs(3)
	clientIp := "192.168.3.100" // matches lans[2]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = findByIpOld(lans, clientIp)
	}
}

func BenchmarkFindByIp_New_ThreeLANs_Last(b *testing.B) {
	lans := makeBenchLANs(3)
	r := newRegistryBench(lans)
	clientIp := "192.168.3.100"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = findByIpNew(r, clientIp)
	}
}

// BenchmarkFindByIp_*_TenLANs_NotFound: 10 LANs, IP not in any LAN (worst case)
func BenchmarkFindByIp_Old_TenLANs_NotFound(b *testing.B) {
	lans := makeBenchLANs(10)
	clientIp := "10.0.0.100" // not in any 192.168.x.0/24
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = findByIpOld(lans, clientIp)
	}
}

func BenchmarkFindByIp_New_TenLANs_NotFound(b *testing.B) {
	lans := makeBenchLANs(10)
	r := newRegistryBench(lans)
	clientIp := "10.0.0.100"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = findByIpNew(r, clientIp)
	}
}

// BenchmarkFindByIp_*_Concurrent: 16 goroutines, 3 LANs — simulates real HTTP traffic
func BenchmarkFindByIp_Old_Concurrent(b *testing.B) {
	lans := makeBenchLANs(3)
	clientIp := "192.168.2.50"
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = findByIpOld(lans, clientIp)
		}
	})
}

func BenchmarkFindByIp_New_Concurrent(b *testing.B) {
	lans := makeBenchLANs(3)
	r := newRegistryBench(lans)
	clientIp := "192.168.2.50"
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = findByIpNew(r, clientIp)
		}
	})
}

// ============================================================================
// BENCHMARKS — GetLanCount: Old (sync.Map.Range) vs New (cached int)
// ============================================================================

// getLanCountOld mirrors current GetLanCount() using sync.Map.Range.
func getLanCountOld(m *sync.Map) int {
	count := 0
	m.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

func BenchmarkGetLanCount_Old(b *testing.B) {
	m := &sync.Map{}
	lans := makeBenchLANs(3)
	for _, l := range lans {
		m.Store(l.name, l.lan)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getLanCountOld(m)
	}
}

func BenchmarkGetLanCount_New(b *testing.B) {
	lans := makeBenchLANs(3)
	r := newRegistryBench(lans)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.mu.RLock()
		_ = r.count
		r.mu.RUnlock()
	}
}

// ============================================================================
// BENCHMARKS — FindAll: Old (sync.Map.Range) vs New (slice copy)
// ============================================================================

func BenchmarkFindAll_Old(b *testing.B) {
	m := &sync.Map{}
	lans := makeBenchLANs(3)
	for _, l := range lans {
		m.Store(l.name, l.lan)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := []*NetworkLan{}
		m.Range(func(key, value any) bool {
			result = append(result, value.(*NetworkLan))
			return true
		})
		_ = result
	}
}

func BenchmarkFindAll_New(b *testing.B) {
	lans := makeBenchLANs(3)
	r := newRegistryBench(lans)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.mu.RLock()
		result := make([]*NetworkLan, 0, r.count)
		for _, entry := range r.byIp {
			result = append(result, entry.lan)
		}
		r.mu.RUnlock()
		_ = result
	}
}
