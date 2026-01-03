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
