package tc

import (
	"errors"
	"fmt"
	"math"
	"net/netip"
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

// Returns the lowest possible value in a given ip segment
func (ipsg *ipsegmt) segMinVal(segIndex int) int {
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

// Get the host bit mask in a given ip segment.
// If segment is not host-masked, it returns 0
func (ipsg *ipsegmt) segMask(segIndex int) int {
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

// Get the hex host mask for a given segment index
func (ipsg *ipsegmt) segMaskHex(segIndex int) (hex string) {
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

// Returns the network ip for the given ip and netmask.
// TODO: Return different format for ipv6
func (ipsg *ipsegmt) baseIp() (ip string) {
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
