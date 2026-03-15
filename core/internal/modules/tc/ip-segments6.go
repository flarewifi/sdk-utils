package tc

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

// ipsegmt6 holds a parsed IPv6 address and prefix length for TC filter
// hash-table bucket calculations.  IPv6 addresses are 128 bits (16 bytes),
// so we operate on 16 segments of 8 bits each, mirroring the ipsegmt
// design for IPv4 but extended to the full 128-bit address space.
type ipsegmt6 struct {
	addr      netip.Addr
	prefixLen int
	segments  []byte // 16 bytes, big-endian
}

// hostMasked6 returns true if segment segIndex (0-15) contains host bits,
// i.e. if the network prefix does not cover the entire segment.
func (ipsg *ipsegmt6) hostMasked6(segIndex int) bool {
	bitEnd := (segIndex * 8) + 8
	return bitEnd > ipsg.prefixLen
}

// segVal6 returns the numeric byte value of segment segIndex.
func (ipsg *ipsegmt6) segVal6(segIndex int) int {
	return int(ipsg.segments[segIndex])
}

// segMask6 returns the number of host bits in segment segIndex.
//
// Returns 0 for segments that are entirely within the network prefix.
// Returns 8 for segments that are entirely within the host portion.
// Returns 1-7 for the single boundary segment that straddles the prefix.
//
// Note: when prefixLen is a multiple of 8, no single boundary segment exists —
// segMask6 returns 0 for the last network byte and 8 for the first host byte,
// which is correct (the boundary falls exactly between two bytes).
func (ipsg *ipsegmt6) segMask6(segIndex int) int {
	startSegIndex := ipsg.prefixLen / 8
	if segIndex > startSegIndex {
		return 8
	}
	if segIndex < startSegIndex {
		return 0
	}
	// segIndex == startSegIndex: this byte straddles the network/host boundary.
	// hostBits is how many bits of this byte belong to the host portion.
	// When prefixLen is a multiple of 8, prefixLen%8 == 0 → hostBits == 8,
	// but startSegIndex already points to the first fully-host byte (not the
	// boundary byte), so returning 8 here is correct.
	return 8 - (ipsg.prefixLen % 8)
}

// segMaxVal6 returns the maximum byte value possible within the subnet for
// segment segIndex. Returns 0 for fully network-masked segments.
func (ipsg *ipsegmt6) segMaxVal6(segIndex int) int {
	if !ipsg.hostMasked6(segIndex) {
		return 0
	}
	hostBits := ipsg.segMask6(segIndex)
	subnetDenom := 1 << hostBits
	segVal := ipsg.segVal6(segIndex)
	subnetIndex := segVal / subnetDenom
	start := subnetIndex * subnetDenom
	return start + subnetDenom - 1
}

// segMinVal6 returns the minimum byte value possible within the subnet for
// segment segIndex. Returns 0 for fully network-masked segments.
func (ipsg *ipsegmt6) segMinVal6(segIndex int) int {
	if !ipsg.hostMasked6(segIndex) {
		return 0
	}
	hostBits := ipsg.segMask6(segIndex)
	subnetDenom := 1 << hostBits
	segVal := ipsg.segVal6(segIndex)
	subnetIndex := segVal / subnetDenom
	return subnetIndex * subnetDenom
}

// segMaskHex6 returns the bitmask for a given segment as a 32-hex-char string
// (16 bytes × 2 hex digits each) suitable for use in tc u32 match commands.
// Only the byte at segIndex is non-zero; all other bytes are "00".
func (ipsg *ipsegmt6) segMaskHex6(segIndex int) string {
	var b strings.Builder
	b.WriteString("0x")
	for i := range ipsg.segments {
		if i != segIndex {
			b.WriteString("00")
		} else {
			mask := (1 << ipsg.segMask6(i)) - 1
			fmt.Fprintf(&b, "%02x", mask)
		}
	}
	return b.String()
}

// baseIp6 returns the network base address in full colon-hex notation (no
// zero-compression), zeroing out all host bits.  tc u32 requires the full
// form rather than RFC 5952 shortened notation.
func (ipsg *ipsegmt6) baseIp6() string {
	masked := make([]byte, 16)
	copy(masked, ipsg.segments)

	for i := range masked {
		if !ipsg.hostMasked6(i) {
			// Byte is entirely within the network prefix — keep it unchanged.
			continue
		}
		hostBits := ipsg.segMask6(i)
		if hostBits >= 8 {
			// Byte is entirely within the host portion — zero it out.
			masked[i] = 0
			continue
		}
		// Byte straddles the boundary: zero out only the host bits.
		// hostBits is 1-7, so (1 << hostBits) never overflows a byte.
		hostMask := byte((1 << hostBits) - 1)
		masked[i] &^= hostMask // clear host bits, keep network bits
	}

	// Build full colon-hex notation (no shortening — tc needs the full form)
	var b strings.Builder
	for i := 0; i < 16; i += 2 {
		if i > 0 {
			b.WriteByte(':')
		}
		fmt.Fprintf(&b, "%02x%02x", masked[i], masked[i+1])
	}
	return b.String()
}

// maskPosition6 returns the byte offset of segIndex within the IPv6 header
// for use in the tc u32 "at" parameter.  The offset is relative to the
// start of the IP (layer-3) header, which is how tc u32 interprets "at".
//
// IPv6 header layout (RFC 8200):
//
//	Bytes  0-3:   Version + Traffic Class + Flow Label
//	Bytes  4-5:   Payload Length
//	Byte   6:     Next Header
//	Byte   7:     Hop Limit
//	Bytes  8-23:  Source Address (16 bytes)
//	Bytes 24-39:  Destination Address (16 bytes)
func maskPosition6(segIndex int, field TcIpField) uint8 {
	if field == TcIpFieldSrc {
		return uint8(8 + segIndex)
	}
	return uint8(24 + segIndex)
}

func newIpsegmt6(ip string, prefixLen int) (*ipsegmt6, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil, err
	}

	if !addr.IsValid() || addr.IsUnspecified() {
		return nil, errors.New("invalid IPv6 address: " + ip)
	}

	// Ensure it is actually an IPv6 address
	if addr.Is4() || addr.Is4In6() {
		return nil, errors.New("address is IPv4, not IPv6: " + ip)
	}

	if prefixLen < 0 || prefixLen > 128 {
		return nil, fmt.Errorf("invalid IPv6 prefix length %d (must be 0-128)", prefixLen)
	}

	raw := addr.As16()
	segments := make([]byte, 16)
	copy(segments, raw[:])

	return &ipsegmt6{
		addr:      addr,
		prefixLen: prefixLen,
		segments:  segments,
	}, nil
}
