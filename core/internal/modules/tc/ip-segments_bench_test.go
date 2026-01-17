package tc

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// ============================================================================
// OLD IMPLEMENTATIONS (Before optimization)
// ============================================================================

func (ipsg *ipsegmt) segMaxValOld(segIndex int) int {
	if ipsg.hostMasked(segIndex) {
		hostbits := int(ipsg.segMask(segIndex))
		subnetDenom := 2
		i := 1
		for i < hostbits {
			subnetDenom += int(math.Pow(float64(2), float64(i)))
			i++
		}

		segval := ipsg.segVal(segIndex)
		subnetIndex := int(math.Floor(float64(segval) / float64(subnetDenom)))
		start := subnetIndex * subnetDenom

		return start + subnetDenom - 1
	}

	return 0
}

func (ipsg *ipsegmt) segMinValOld(segIndex int) int {
	if ipsg.hostMasked(segIndex) {
		hostbits := ipsg.segMask(segIndex)
		subnetDenom := 2
		i := 1
		for i < int(hostbits) {
			subnetDenom += int(math.Pow(float64(2), float64(i)))
			i++
		}

		segval := ipsg.segVal(segIndex)
		subnetIndex := int(math.Floor(float64(segval) / float64(subnetDenom)))
		start := subnetIndex * subnetDenom
		return start
	}

	return 0
}

func (ipsg *ipsegmt) segMaskOld(segIndex int) int {
	startSegIndex := int(math.Ceil(float64(ipsg.netmask+1)/8)) - 1
	if segIndex > startSegIndex {
		return 8
	}

	if segIndex < startSegIndex {
		return 0
	}

	hostmask := 8 - (ipsg.netmask % 8)
	return hostmask
}

func (ipsg *ipsegmt) segMaskHexOld(segIndex int) (hex string) {
	hex = "0x"
	i := 0
	for i < len(ipsg.segments) {
		if i != segIndex {
			hex += "00"
		} else {
			mask := int(math.Pow(float64(2), float64(ipsg.segMask(i)))) - 1
			hex = fmt.Sprintf("%s%02x", hex, mask)
		}
		i++
	}
	return hex
}

func (ipsg *ipsegmt) baseIpOld() (ip string) {
	segIndex := 0
	count := len(ipsg.segments)
	for segIndex < count {
		if ipsg.hostMasked(segIndex) {
			ip += fmt.Sprintf("%d", ipsg.segMinVal(segIndex))
		} else {
			ip += fmt.Sprintf("%d", ipsg.segVal(segIndex))
		}
		if segIndex < count-1 {
			ip += "."
		}
		segIndex++
	}

	return ip
}

// ============================================================================
// BENCHMARKS - New vs Old
// ============================================================================

func BenchmarkSegMaxVal_New(b *testing.B) {
	ipsg, _ := newIpsegmt("10.0.17.100", 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ipsg.segMaxVal(2)
	}
}

func BenchmarkSegMaxVal_Old(b *testing.B) {
	ipsg, _ := newIpsegmt("10.0.17.100", 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ipsg.segMaxValOld(2)
	}
}

func BenchmarkSegMinVal_New(b *testing.B) {
	ipsg, _ := newIpsegmt("10.0.17.100", 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ipsg.segMinVal(2)
	}
}

func BenchmarkSegMinVal_Old(b *testing.B) {
	ipsg, _ := newIpsegmt("10.0.17.100", 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ipsg.segMinValOld(2)
	}
}

func BenchmarkSegMask_New(b *testing.B) {
	ipsg, _ := newIpsegmt("10.0.17.100", 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ipsg.segMask(2)
	}
}

func BenchmarkSegMask_Old(b *testing.B) {
	ipsg, _ := newIpsegmt("10.0.17.100", 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ipsg.segMaskOld(2)
	}
}

func BenchmarkSegMaskHex_New(b *testing.B) {
	ipsg, _ := newIpsegmt("10.0.17.100", 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ipsg.segMaskHex(2)
	}
}

func BenchmarkSegMaskHex_Old(b *testing.B) {
	ipsg, _ := newIpsegmt("10.0.17.100", 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ipsg.segMaskHexOld(2)
	}
}

func BenchmarkBaseIp_New(b *testing.B) {
	ipsg, _ := newIpsegmt("192.168.1.100", 24)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ipsg.baseIp()
	}
}

func BenchmarkBaseIp_Old(b *testing.B) {
	ipsg, _ := newIpsegmt("192.168.1.100", 24)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ipsg.baseIpOld()
	}
}

// ============================================================================
// BENCHMARKS - Real-world scenarios (nested loops as in tc-filter.go)
// ============================================================================

func BenchmarkRealWorld_FilterSetup_New(b *testing.B) {
	ipsg, _ := newIpsegmt("192.168.1.1", 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := len(ipsg.segments)
		for segIndex := 0; segIndex < count; segIndex++ {
			if ipsg.hostMasked(segIndex) {
				_ = ipsg.segMaxVal(segIndex) + 1
				_ = ipsg.segMaskHex(segIndex)
				_ = ipsg.baseIp()

				parentSegIndex := segIndex - 1
				if segIndex > 0 && ipsg.hostMasked(parentSegIndex) {
					listIndex := ipsg.segMinVal(parentSegIndex)
					maxIndex := ipsg.segMaxVal(parentSegIndex)
					for j := listIndex; j <= maxIndex; j++ {
						// Simulate tc filter commands
						_ = j
					}
				}
			}
		}
	}
}

func BenchmarkRealWorld_FilterSetup_Old(b *testing.B) {
	ipsg, _ := newIpsegmt("192.168.1.1", 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := len(ipsg.segments)
		for segIndex := 0; segIndex < count; segIndex++ {
			if ipsg.hostMasked(segIndex) {
				_ = ipsg.segMaxValOld(segIndex) + 1
				_ = ipsg.segMaskHexOld(segIndex)
				_ = ipsg.baseIpOld()

				parentSegIndex := segIndex - 1
				if segIndex > 0 && ipsg.hostMasked(parentSegIndex) {
					listIndex := ipsg.segMinValOld(parentSegIndex)
					maxIndex := ipsg.segMaxValOld(parentSegIndex)
					for j := listIndex; j <= maxIndex; j++ {
						// Simulate tc filter commands
						_ = j
					}
				}
			}
		}
	}
}

// ============================================================================
// BENCHMARKS - strings.Builder vs concatenation
// ============================================================================

func BenchmarkStringBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var builder strings.Builder
		builder.WriteString("192")
		builder.WriteByte('.')
		builder.WriteString("168")
		builder.WriteByte('.')
		builder.WriteString("1")
		builder.WriteByte('.')
		builder.WriteString("0")
		_ = builder.String()
	}
}

func BenchmarkStringConcatenation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s := ""
		s += "192"
		s += "."
		s += "168"
		s += "."
		s += "1"
		s += "."
		s += "0"
		_ = s
	}
}

func BenchmarkBitshiftPower(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = 1 << 4 // 2^4 = 16
	}
}

func BenchmarkMathPowPower(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = int(math.Pow(2, 4))
	}
}
