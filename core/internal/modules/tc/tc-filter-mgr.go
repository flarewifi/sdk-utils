package tc

import (
	"net"
)

// TcFilterMgr manages TC filters for a single LAN device, supporting both
// IPv4 (TcFilter) and IPv6 (TcFilter6) clients transparently.
type TcFilterMgr struct {
	dev       string
	tcFilter4 *TcFilter  // IPv4 filter — nil until Setup() is called with an IPv4 addr
	tcFilter6 *TcFilter6 // IPv6 filter — nil until Setup6() is called with an IPv6 addr
}

func NewTcFilterMgr(dev string) *TcFilterMgr {
	return &TcFilterMgr{
		dev: dev,
	}
}

// Setup initialises the IPv4 TC hash filter for the given network address and
// prefix length.  It must be called before CreateFilter/DeleteFilter for IPv4.
func (self *TcFilterMgr) Setup(ip string, netmask int) (err error) {
	filter, err := NewTcFilter(self.dev, ip, netmask)
	if err != nil {
		return err
	}
	if err := filter.Setup(); err != nil {
		return err
	}

	self.tcFilter4 = filter
	return nil
}

// Setup6 initialises the IPv6 TC hash filter for the given network address and
// prefix length.  It must be called before CreateFilter/DeleteFilter for IPv6.
func (self *TcFilterMgr) Setup6(ip string, prefixLen int) (err error) {
	filter, err := NewTcFilter6(self.dev, ip, prefixLen)
	if err != nil {
		return err
	}
	if err := filter.Setup(); err != nil {
		return err
	}

	self.tcFilter6 = filter
	return nil
}

// Reset re-initialises both IPv4 and IPv6 filters, restoring all active
// client filter entries.  Either filter is skipped if not yet initialised.
func (self *TcFilterMgr) Reset() (err error) {
	if self.tcFilter4 != nil {
		if err = self.tcFilter4.Reset(); err != nil {
			return err
		}
	}
	if self.tcFilter6 != nil {
		if err = self.tcFilter6.Reset(); err != nil {
			return err
		}
	}
	return nil
}

// CreateFilter adds a TC filter for clientIp (IPv4 or IPv6) pointing to classid.
// The correct underlying filter (v4 or v6) is chosen automatically.
func (self *TcFilterMgr) CreateFilter(clientIp string, classid TcClassId) error {
	if isIPv6Address(clientIp) {
		if self.tcFilter6 == nil {
			// IPv6 filter not set up — no-op (interface may not have IPv6)
			return nil
		}
		return self.tcFilter6.CreateFilter(clientIp, classid.String())
	}
	if self.tcFilter4 == nil {
		return nil
	}
	return self.tcFilter4.CreateFilter(clientIp, classid.String())
}

// DeleteFilter removes the TC filter for clientIp (IPv4 or IPv6).
func (self *TcFilterMgr) DeleteFilter(clientIp string) error {
	if isIPv6Address(clientIp) {
		if self.tcFilter6 == nil {
			return nil
		}
		return self.tcFilter6.DeleteFilter(clientIp)
	}
	if self.tcFilter4 == nil {
		return nil
	}
	return self.tcFilter4.DeleteFilter(clientIp)
}

// CleanUp removes all TC filters (both IPv4 and IPv6) from the device.
func (self *TcFilterMgr) CleanUp() error {
	if self.tcFilter4 != nil {
		if err := self.tcFilter4.CleanUp(); err != nil {
			return err
		}
	}
	if self.tcFilter6 != nil {
		if err := self.tcFilter6.CleanUp(); err != nil {
			return err
		}
	}
	return nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// isIPv6Address returns true if ip is a valid IPv6 address (not IPv4 or
// IPv4-mapped).  Used to route filter operations to the correct manager.
func isIPv6Address(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.To4() == nil
}
