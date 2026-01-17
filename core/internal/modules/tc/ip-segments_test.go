package tc

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_newIpsegts(t *testing.T) {
	var err error
	_, err = newIpsegmt("xxxxxx", 0)
	assert.NotNil(t, err)

	_, err = newIpsegmt("10.0.0.1", 20)
	assert.Nil(t, err)
}

func Test_ipsegts_hostMasked(t *testing.T) {
	ipsg, _ := newIpsegmt("192.168.1.1", 24)

	assert.False(t, ipsg.hostMasked(0))
	assert.False(t, ipsg.hostMasked(1))
	assert.False(t, ipsg.hostMasked(2))
	assert.True(t, ipsg.hostMasked(3))

	ipsg.netmask = 23

	assert.False(t, ipsg.hostMasked(0))
	assert.False(t, ipsg.hostMasked(1))
	assert.True(t, ipsg.hostMasked(2))
	assert.True(t, ipsg.hostMasked(3))
}

func Test_ipsegts_segmtVal(t *testing.T) {
	ipsg, _ := newIpsegmt("192.168.222.123", 24)

	assert.Equal(t, 192, ipsg.segVal(0))
	assert.Equal(t, 168, ipsg.segVal(1))
	assert.Equal(t, 222, ipsg.segVal(2))
	assert.Equal(t, 123, ipsg.segVal(3))

	ipsg, _ = newIpsegmt("10.0.0.1", 20)
	assert.Equal(t, 10, ipsg.segVal(0))
	assert.Equal(t, 0, ipsg.segVal(1))
	assert.Equal(t, 0, ipsg.segVal(2))
	assert.Equal(t, 1, ipsg.segVal(3))
}

func Test_ipsegts_segmtMask(t *testing.T) {
	ipsg, _ := newIpsegmt("192.168.222.123", 24)
	assert.Equal(t, 0, ipsg.segMask(0))
	assert.Equal(t, 0, ipsg.segMask(1))
	assert.Equal(t, 0, ipsg.segMask(2))
	assert.Equal(t, 8, ipsg.segMask(3))

	ipsg, _ = newIpsegmt("192.168.222.123", 25)
	assert.Equal(t, 0, ipsg.segMask(0))
	assert.Equal(t, 0, ipsg.segMask(1))
	assert.Equal(t, 0, ipsg.segMask(2))
	assert.Equal(t, 7, ipsg.segMask(3))

	ipsg, _ = newIpsegmt("10.0.0.1", 20)
	assert.Equal(t, 0, ipsg.segMask(0))
	assert.Equal(t, 0, ipsg.segMask(1))
	assert.Equal(t, 4, ipsg.segMask(2))
	assert.Equal(t, 8, ipsg.segMask(3))

	ipsg, _ = newIpsegmt("10.0.0.1", 23)
	assert.Equal(t, 0, ipsg.segMask(0))
	assert.Equal(t, 0, ipsg.segMask(1))
	assert.Equal(t, 1, ipsg.segMask(2))
	assert.Equal(t, 8, ipsg.segMask(3))
}

func Test_ipsegts_segmtMaskHex(t *testing.T) {
	ipsg, _ := newIpsegmt("192.168.222.123", 24)
	assert.Equal(t, "0x00000000", ipsg.segMaskHex(0))
	assert.Equal(t, "0x00000000", ipsg.segMaskHex(1))
	assert.Equal(t, "0x00000000", ipsg.segMaskHex(2))
	assert.Equal(t, "0x000000ff", ipsg.segMaskHex(3))

	ipsg, _ = newIpsegmt("192.168.222.123", 25)
	assert.Equal(t, "0x00000000", ipsg.segMaskHex(0))
	assert.Equal(t, "0x00000000", ipsg.segMaskHex(1))
	assert.Equal(t, "0x00000000", ipsg.segMaskHex(2))
	assert.Equal(t, "0x0000007f", ipsg.segMaskHex(3))

	ipsg, _ = newIpsegmt("10.0.0.1", 20)
	assert.Equal(t, "0x00000000", ipsg.segMaskHex(0))
	assert.Equal(t, "0x00000000", ipsg.segMaskHex(1))
	assert.Equal(t, "0x00000f00", ipsg.segMaskHex(2))
	assert.Equal(t, "0x000000ff", ipsg.segMaskHex(3))

	ipsg, _ = newIpsegmt("10.0.0.1", 23)
	assert.Equal(t, "0x00000000", ipsg.segMaskHex(0))
	assert.Equal(t, "0x00000000", ipsg.segMaskHex(1))
	assert.Equal(t, "0x00000100", ipsg.segMaskHex(2))
	assert.Equal(t, "0x000000ff", ipsg.segMaskHex(3))
}

func Test_ipsegts_segMaxVal(t *testing.T) {
	ipsg, _ := newIpsegmt("10.0.0.1", 20)
	assert.Equal(t, 0, ipsg.segMaxVal(0))
	assert.Equal(t, 0, ipsg.segMaxVal(1))
	assert.Equal(t, 15, ipsg.segMaxVal(2))
	assert.Equal(t, 255, ipsg.segMaxVal(3))

	ipsg, _ = newIpsegmt("10.0.16.1", 20)
	assert.Equal(t, 0, ipsg.segMaxVal(0))
	assert.Equal(t, 0, ipsg.segMaxVal(1))
	assert.Equal(t, 31, ipsg.segMaxVal(2))
	assert.Equal(t, 255, ipsg.segMaxVal(3))

	ipsg, _ = newIpsegmt("10.0.3.1", 23)
	assert.Equal(t, 0, ipsg.segMaxVal(0))
	assert.Equal(t, 0, ipsg.segMaxVal(1))
	assert.Equal(t, 3, ipsg.segMaxVal(2))
	assert.Equal(t, 255, ipsg.segMaxVal(3))

	ipsg, _ = newIpsegmt("192.168.1.1", 24)
	assert.Equal(t, 0, ipsg.segMaxVal(0))
	assert.Equal(t, 0, ipsg.segMaxVal(1))
	assert.Equal(t, 0, ipsg.segMaxVal(2))
	assert.Equal(t, 255, ipsg.segMaxVal(3))
}

func Test_ipsegts_segMinVal(t *testing.T) {
	ipsg, _ := newIpsegmt("10.0.17.100", 20)
	assert.Equal(t, 0, ipsg.segMinVal(0))
	assert.Equal(t, 0, ipsg.segMinVal(1))
	assert.Equal(t, 16, ipsg.segMinVal(2))
	assert.Equal(t, 0, ipsg.segMinVal(3))

	ipsg, _ = newIpsegmt("10.0.15.100", 20)
	assert.Equal(t, 0, ipsg.segMinVal(0))
	assert.Equal(t, 0, ipsg.segMinVal(1))
	assert.Equal(t, 0, ipsg.segMinVal(2))
	assert.Equal(t, 0, ipsg.segMinVal(3))
}

func Test_ipsegts_baseIp(t *testing.T) {
	// Standard /24 network
	ipsg, _ := newIpsegmt("192.168.1.100", 24)
	assert.Equal(t, "192.168.1.0", ipsg.baseIp())

	// /20 network with host bits in 3rd octet
	ipsg, _ = newIpsegmt("10.0.17.100", 20)
	assert.Equal(t, "10.0.16.0", ipsg.baseIp())

	// /23 network
	ipsg, _ = newIpsegmt("172.16.5.50", 23)
	assert.Equal(t, "172.16.4.0", ipsg.baseIp())

	// /25 network (split last octet)
	ipsg, _ = newIpsegmt("192.168.1.200", 25)
	assert.Equal(t, "192.168.1.128", ipsg.baseIp())

	// /32 network (single host)
	ipsg, _ = newIpsegmt("10.0.0.1", 32)
	assert.Equal(t, "10.0.0.1", ipsg.baseIp())

	// /8 network (class A)
	ipsg, _ = newIpsegmt("10.50.100.200", 8)
	assert.Equal(t, "10.0.0.0", ipsg.baseIp())

	// /16 network (class B)
	ipsg, _ = newIpsegmt("172.16.50.100", 16)
	assert.Equal(t, "172.16.0.0", ipsg.baseIp())

	// Edge case: IP at subnet boundary (start)
	ipsg, _ = newIpsegmt("10.0.16.0", 20)
	assert.Equal(t, "10.0.16.0", ipsg.baseIp())

	// Edge case: IP at subnet boundary (end)
	ipsg, _ = newIpsegmt("10.0.31.255", 20)
	assert.Equal(t, "10.0.16.0", ipsg.baseIp())
}

func Test_ipsegts_EdgeCases(t *testing.T) {
	// /32 - single host network
	ipsg, _ := newIpsegmt("10.0.0.1", 32)
	assert.Equal(t, 0, ipsg.segMask(0))
	assert.Equal(t, 0, ipsg.segMask(1))
	assert.Equal(t, 0, ipsg.segMask(2))
	assert.Equal(t, 0, ipsg.segMask(3))
	// With /32, no host bits in any segment
	assert.False(t, ipsg.hostMasked(0))
	assert.False(t, ipsg.hostMasked(1))
	assert.False(t, ipsg.hostMasked(2))
	assert.False(t, ipsg.hostMasked(3))

	// /8 - class A network
	ipsg, _ = newIpsegmt("10.50.100.200", 8)
	assert.Equal(t, 0, ipsg.segMask(0))
	assert.Equal(t, 8, ipsg.segMask(1))
	assert.Equal(t, 8, ipsg.segMask(2))
	assert.Equal(t, 8, ipsg.segMask(3))
	assert.Equal(t, 255, ipsg.segMaxVal(1))
	assert.Equal(t, 255, ipsg.segMaxVal(2))
	assert.Equal(t, 255, ipsg.segMaxVal(3))
	assert.Equal(t, 0, ipsg.segMinVal(1))
	assert.Equal(t, 0, ipsg.segMinVal(2))
	assert.Equal(t, 0, ipsg.segMinVal(3))

	// /16 - class B network
	ipsg, _ = newIpsegmt("172.16.50.100", 16)
	assert.Equal(t, 0, ipsg.segMask(0))
	assert.Equal(t, 0, ipsg.segMask(1))
	assert.Equal(t, 8, ipsg.segMask(2))
	assert.Equal(t, 8, ipsg.segMask(3))
	assert.Equal(t, 255, ipsg.segMaxVal(2))
	assert.Equal(t, 255, ipsg.segMaxVal(3))
	assert.Equal(t, 0, ipsg.segMinVal(2))
	assert.Equal(t, 0, ipsg.segMinVal(3))

	// Boundary IP (at subnet start)
	ipsg, _ = newIpsegmt("10.0.16.0", 20)
	assert.Equal(t, 16, ipsg.segMinVal(2))
	assert.Equal(t, 31, ipsg.segMaxVal(2))
	assert.Equal(t, "10.0.16.0", ipsg.baseIp())

	// Boundary IP (at subnet end)
	ipsg, _ = newIpsegmt("10.0.31.255", 20)
	assert.Equal(t, 16, ipsg.segMinVal(2))
	assert.Equal(t, 31, ipsg.segMaxVal(2))
	assert.Equal(t, "10.0.16.0", ipsg.baseIp())

	// /1 - extreme case (half the internet)
	ipsg, _ = newIpsegmt("128.0.0.1", 1)
	assert.Equal(t, 7, ipsg.segMask(0))
	assert.True(t, ipsg.hostMasked(0))
	assert.Equal(t, 128, ipsg.segMinVal(0))
	assert.Equal(t, 255, ipsg.segMaxVal(0))

	// /31 - point-to-point link (2 hosts)
	ipsg, _ = newIpsegmt("10.0.0.1", 31)
	assert.Equal(t, 1, ipsg.segMask(3))
	assert.Equal(t, 0, ipsg.segMinVal(3))
	assert.Equal(t, 1, ipsg.segMaxVal(3))
}
