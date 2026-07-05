package network

import (
	"errors"
	"sync"

	"core/internal/modules/tc"
	"core/internal/modules/ubus"
	"core/utils/config"
	jobque "core/utils/job-que"
)

var (
	networkQue = jobque.NewJobQueue[any]()

	// errTcNotEnabled guards the per-client TC methods on a LAN that is registered
	// (for client identification) but not captive, so it has no TC managers. The
	// session flow only ever targets captive LANs, so this is defensive — it turns
	// a would-be nil dereference into a clean error.
	errTcNotEnabled = errors.New("traffic control is not enabled on this interface")
)

type NetworkLan struct {
	mu          sync.RWMutex
	name        string
	up          bool
	tcClassMgr  *tc.TcClassMgr
	tcFilterMgr *tc.TcFilterMgr
}

func NewNetworkLan(ifname string) *NetworkLan {
	return &NetworkLan{
		name: ifname,
		up:   true,
	}
}

func (self *NetworkLan) Name() string {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.name
}

func (self *NetworkLan) Bandwidth() (download tc.Mbit, upload tc.Mbit) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	if self.tcClassMgr == nil {
		return 0, 0
	}
	d, u := self.tcClassMgr.Bandwidth()
	return d.ToMbit(), u.ToMbit()
}

// HasTrafficControl reports whether TC is set up on this LAN (i.e. it is a
// captive interface). Used by the reconcile to decide setup vs teardown.
func (self *NetworkLan) HasTrafficControl() bool {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.tcClassMgr != nil
}

// TeardownTrafficControl removes all TC qdiscs/classes/filters for this LAN and
// drops the managers, returning the interface to an unshaped state. Best-effort;
// used when an interface is switched from captive to free.
func (self *NetworkLan) TeardownTrafficControl() {
	networkQue.Exec("TeardownTrafficControl", func() (any, error) {
		self.mu.Lock()
		classMgr := self.tcClassMgr
		filterMgr := self.tcFilterMgr
		self.tcClassMgr = nil
		self.tcFilterMgr = nil
		self.mu.Unlock()

		if filterMgr != nil {
			filterMgr.CleanUp()
		}
		if classMgr != nil {
			classMgr.CleanUp()
		}
		return nil, nil
	})
}

func (self *NetworkLan) ResetTc() (err error) {
	_, err = networkQue.Exec("ResetTc", func() (any, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()

		if self.tcClassMgr == nil || self.tcFilterMgr == nil {
			return nil, errors.New("TC managers not initialized")
		}

		err = self.tcClassMgr.Reset()
		if err != nil {
			return nil, err
		}

		err = self.tcFilterMgr.Reset()
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

// ReinitializeTc completely reinitializes TC (classes, filters, and captive portal)
// This is used when the network interface comes back up after being down
// IMPORTANT: Preserves all active session TC classes and filters
func (self *NetworkLan) ReinitializeTc() (err error) {
	_, err = networkQue.Exec("ReinitializeTc", func() (any, error) {
		// Get reference to existing TC managers to preserve session data.
		// NOTE: The lock is released before subsequent operations, but this is safe
		// because all TC operations are serialized through networkQue. No concurrent
		// modifications to tcClassMgr/tcFilterMgr can occur while this function runs.
		self.mu.RLock()
		oldClassMgr := self.tcClassMgr
		oldFilterMgr := self.tcFilterMgr
		self.mu.RUnlock()

		// Get fresh interface info from UBUS
		cfg, err := config.ReadBandwidthConfig()
		if err != nil {
			return nil, err
		}

		i, err := ubus.GetNetworkInterface(self.name)
		if err != nil {
			return nil, err
		}

		// No bandwidth.json entry → default to global bandwidth (LanCfg), and the
		// 0 global speed drives auto-detect below, so a captive interface still
		// shapes even without an explicit cap.
		lanCfg := cfg.LanCfg(self.name)

		dev := i.Device

		// Auto-detect link speed when configured speed is 0
		if lanCfg.GlobalDownMbits == 0 || lanCfg.GlobalUpMbits == 0 {
			detectedSpeed := defaultSpeed
			netDev, err := ubus.GetNetworkDevice(dev)
			if err == nil && netDev != nil {
				detectedSpeed = ParseLinkSpeed(netDev.Speed)
			}

			if lanCfg.GlobalDownMbits == 0 {
				lanCfg.GlobalDownMbits = detectedSpeed
			}
			if lanCfg.GlobalUpMbits == 0 {
				lanCfg.GlobalUpMbits = detectedSpeed
			}
		}
		ipv4, err := i.IpV4Addr()
		if err != nil {
			return nil, err
		}

		// If TC managers exist, use Reset() to preserve session data
		// Otherwise, create new managers
		if oldClassMgr != nil && oldFilterMgr != nil {
			// Reset TC Class Manager (preserves classList)
			err = oldClassMgr.Reset()
			if err != nil {
				return nil, err
			}

			// Reset TC Filter Manager (preserves filterList)
			err = oldFilterMgr.Reset()
			if err != nil {
				return nil, err
			}
		} else {
			// First time setup or managers were nil

			// Setup TC Class Manager
			classMgr := tc.NewTcClassMgr(dev, tc.Kbit(lanCfg.GlobalDownMbits*1000), tc.Kbit(lanCfg.GlobalUpMbits*1000))
			err = classMgr.Setup()
			if err != nil {
				return nil, err
			}

			self.mu.Lock()
			self.tcClassMgr = classMgr
			self.mu.Unlock()

			// Setup TC Filter Manager (IPv4)
			filterMgr := tc.NewTcFilterMgr(i.Device)
			err = filterMgr.Setup(ipv4.Addr, ipv4.Netmask)
			if err != nil {
				return nil, err
			}

			// Setup TC Filter Manager (IPv6) — optional, non-fatal.
			if ipv6, err := NewNetworkInterface(self.name).IpV6Addr(); err == nil {
				filterMgr.Setup6(ipv6.Addr, ipv6.PrefixLen)
			}

			self.mu.Lock()
			self.tcFilterMgr = filterMgr
			self.mu.Unlock()
		}

		// Captive portal + split-horizon DNS are owned by ApplyPortalConfig (see
		// portal-config.go). listenLanEvents re-applies it right after this reinit,
		// so the portal DNAT target / DNS follow the (possibly changed) main IP
		// there rather than being pinned to this interface's own IP here.
		return nil, nil
	})

	return err
}

func (self *NetworkLan) SetupTrafficControl() (err error) {
	_, err = networkQue.Exec("SetupTrafficControl", func() (interface{}, error) {
		cfg, err := config.ReadBandwidthConfig()
		if err != nil {
			return nil, err
		}

		i, err := ubus.GetNetworkInterface(self.name)
		if err != nil {
			return nil, err
		}

		// A captive interface may have no bandwidth.json entry yet; LanCfg then
		// defaults to global bandwidth, and its 0 global speed drives the
		// auto-detect link-speed path below instead of failing setup.
		c := cfg.LanCfg(self.name)
		{
			dev := i.Device

			// Auto-detect link speed when configured speed is 0
			if c.GlobalDownMbits == 0 || c.GlobalUpMbits == 0 {
				detectedSpeed := defaultSpeed
				netDev, err := ubus.GetNetworkDevice(dev)
				if err == nil && netDev != nil {
					detectedSpeed = ParseLinkSpeed(netDev.Speed)
				}

				if c.GlobalDownMbits == 0 {
					c.GlobalDownMbits = detectedSpeed
				}
				if c.GlobalUpMbits == 0 {
					c.GlobalUpMbits = detectedSpeed
				}
			}

			classMgr := tc.NewTcClassMgr(dev, tc.Kbit(c.GlobalDownMbits*1000), tc.Kbit(c.GlobalUpMbits*1000))
			err = classMgr.Setup()
			if err != nil {
				return nil, err
			}

			self.mu.Lock()
			self.tcClassMgr = classMgr
			self.mu.Unlock()

			ipv4, err := i.IpV4Addr()
			if err != nil {
				return nil, err
			}

			filterMgr := tc.NewTcFilterMgr(i.Device)
			err = filterMgr.Setup(ipv4.Addr, ipv4.Netmask)
			if err != nil {
				return nil, err
			}

			// Set up IPv6 TC filter if interface has an IPv6 address.
			if ipv6, err := NewNetworkInterface(self.name).IpV6Addr(); err == nil {
				if err := filterMgr.Setup6(ipv6.Addr, ipv6.PrefixLen); err != nil {
				}
			}

			self.mu.Lock()
			self.tcFilterMgr = filterMgr
			self.mu.Unlock()

			// Captive portal + split-horizon DNS are applied centrally by
			// ApplyPortalConfig once all LANs are set up (see portal-config.go),
			// so traffic control here stays concerned only with bandwidth.
			return nil, nil
		}
	})

	return err
}

func (self *NetworkLan) Up() bool {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.up
}

func (self *NetworkLan) SetStatus(up bool) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.up = up
}

func (self *NetworkLan) GetInterface() *NetworkInterface {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return NewNetworkInterface(self.name)
}

func (self *NetworkLan) CreateClass(classid uint, downMbit int, upMbit int) error {
	_, err := networkQue.Exec("CreateClass", func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()

		if self.tcClassMgr == nil {
			return nil, errTcNotEnabled
		}

		downKbit := tc.Kbit(downMbit * 1000)
		upKbit := tc.Kbit(upMbit * 1000)

		return nil, self.tcClassMgr.CreateClass(self.tcClassMgr.UserTcClass(), tc.TcClassId(classid), 1, 1, downKbit, upKbit)
	})
	return err
}

func (self *NetworkLan) ChangeClass(classid uint, downMbit int, upMbit int) error {
	_, err := networkQue.Exec("ChangeClass", func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()

		if self.tcClassMgr == nil {
			return nil, errTcNotEnabled
		}

		downKbit := tc.Kbit(downMbit * 1000)
		upKbit := tc.Kbit(upMbit * 1000)

		return nil, self.tcClassMgr.ChangeClass(self.tcClassMgr.UserTcClass(), tc.TcClassId(classid), 1, 1, downKbit, upKbit)
	})
	return err
}

func (self *NetworkLan) DelClass(classid uint) error {
	_, err := networkQue.Exec("DelClass", func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()
		if self.tcClassMgr == nil {
			return nil, errTcNotEnabled
		}
		return nil, self.tcClassMgr.DeleteClass(tc.TcClassId(classid))
	})
	return err
}

func (self *NetworkLan) CreateFilter(ip string, classid uint) error {
	_, err := networkQue.Exec("CreateFilter", func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()
		if self.tcFilterMgr == nil {
			return nil, errTcNotEnabled
		}
		return nil, self.tcFilterMgr.CreateFilter(ip, tc.TcClassId(classid))
	})
	return err
}

func (self *NetworkLan) DelFilter(ip string, classid uint) error {
	_, err := networkQue.Exec("DelFilter", func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()
		if self.tcFilterMgr == nil {
			return nil, errTcNotEnabled
		}
		return nil, self.tcFilterMgr.DeleteFilter(ip)
	})
	return err
}

func (self *NetworkLan) UpdateBandwidth(downMbits int, upMbits int) error {
	_, err := networkQue.Exec("UpdateBandwidth", func() (any, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()

		if self.tcClassMgr == nil {
			return nil, errTcNotEnabled
		}

		downKbit := tc.Mbit(downMbits).ToKbit()
		upKbit := tc.Mbit(upMbits).ToKbit()
		return nil, self.tcClassMgr.UpdateBandwidth(downKbit, upKbit)
	})
	return err
}
