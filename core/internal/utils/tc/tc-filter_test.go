package tc

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Tc_Setup(t *testing.T) {
	dev := "br-lan"
	tc, err := NewTcFilter(dev, "10.0.10.11", 20)
	assert.Nil(t, err)
	tc.Setup()
}

func Test_Tc_htBktFor(t *testing.T) {
	dev := "br-lan"
	tc, _ := NewTcFilter(dev, "10.0.10.11", 20)

	htbkt, _ := tc.hashBktFor("10.0.0.1")
	assert.Equal(t, "2:1:", htbkt)

	htbkt, _ = tc.hashBktFor("10.0.15.254")
	assert.Equal(t, "11:fe:", htbkt)

	tc, _ = NewTcFilter(dev, "192.168.0.254", 24)

	htbkt, _ = tc.hashBktFor("192.168.0.1")
	assert.Equal(t, "1:1:", htbkt)

	htbkt, _ = tc.hashBktFor("192.168.0.254")
	assert.Equal(t, "1:fe:", htbkt)
}
