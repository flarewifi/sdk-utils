package tc

import (
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

type ipsegmt struct {
	addr     netip.Addr
	netmask  int
	segments []byte
}

// Check if a given ip segment has host bits
func (ipsg *ipsegmt) hostMasked(segIndex int) bool {
	bitEnd := (segIndex * 8) + 8
	return bitEnd > ipsg.netmask
}

// Returns the numeric value of a given segment
func (ipsg *ipsegmt) segVal(segIndex int) int {
	b := ipsg.segments[segIndex]
	return int(b)
}

// Returns the max possible value of a given ip segment.
// If segment has no host bits, it returns 0
func (ipsg *ipsegmt) segMaxVal(segIndex int) int {
	if ipsg.hostMasked(segIndex) {
		hostbits := ipsg.segMask(segIndex)
		subnetDenom := 1 << hostbits // 2^hostbits

		segval := ipsg.segVal(segIndex)
		subnetIndex := segval / subnetDenom // Integer division already floors
		start := subnetIndex * subnetDenom

		return start + subnetDenom - 1
	}

	return 0
}

// Returns the lowest possible value in a given ip segment
func (ipsg *ipsegmt) segMinVal(segIndex int) int {
	if ipsg.hostMasked(segIndex) {
		hostbits := ipsg.segMask(segIndex)
		subnetDenom := 1 << hostbits // 2^hostbits

		segval := ipsg.segVal(segIndex)
		subnetIndex := segval / subnetDenom // Integer division already floors
		start := subnetIndex * subnetDenom
		return start
	}

	return 0
}

// Get the host bit mask in a given ip segment.
// If segment is not host-masked, it returns 0
func (ipsg *ipsegmt) segMask(segIndex int) int {
	startSegIndex := ipsg.netmask / 8 // Equivalent: ceil((n+1)/8) - 1 = n/8
	if segIndex > startSegIndex {
		return 8
	}

	if segIndex < startSegIndex {
		return 0
	}

	hostmask := 8 - (ipsg.netmask % 8)
	return hostmask
}

// Get the hex host mask for a given segment index
func (ipsg *ipsegmt) segMaskHex(segIndex int) string {
	var b strings.Builder
	b.WriteString("0x")
	for i := range ipsg.segments {
		if i != segIndex {
			b.WriteString("00")
		} else {
			mask := (1 << ipsg.segMask(i)) - 1
			fmt.Fprintf(&b, "%02x", mask)
		}
	}
	return b.String()
}

// Returns the network ip for the given ip and netmask.
// TODO: Return different format for ipv6
func (ipsg *ipsegmt) baseIp() string {
	var b strings.Builder
	for i, seg := range ipsg.segments {
		if i > 0 {
			b.WriteByte('.')
		}
		if ipsg.hostMasked(i) {
			b.WriteString(strconv.Itoa(ipsg.segMinVal(i)))
		} else {
			b.WriteString(strconv.Itoa(int(seg)))
		}
	}
	return b.String()
}

func newIpsegmt(ip string, netmask int) (*ipsegmt, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil, err
	}

	if !addr.IsValid() || addr.IsUnspecified() {
		return nil, errors.New("Invalid IP address " + ip)
	}

	segments := &ipsegmt{
		netmask:  netmask,
		addr:     addr,
		segments: addr.AsSlice(),
	}

	return segments, nil
}
