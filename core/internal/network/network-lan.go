package network

import (
	"errors"
	"sync"

	"core/internal/modules/captivedns"
	"core/internal/modules/nftables"
	"core/internal/modules/tc"
	"core/internal/modules/ubus"
	"core/utils/config"
	jobque "core/utils/job-que"
)

var (
	networkQue = jobque.NewJobQueue[any]()
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
	d, u := self.tcClassMgr.Bandwidth()
	return d.ToMbit(), u.ToMbit()
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

		lanCfg, ok := cfg.Lans[self.name]
		if !ok {
			return nil, errors.New(self.name + " network config not found")
		}

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

		// Declared here so both branches of the if/else below can populate it,
		// and the captive portal setup after the block can read it.
		var reinitIPv6Addr string

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
			// Fetch IPv6 once; reuse for captive portal below.
			if ipv6, err := NewNetworkInterface(self.name).IpV6Addr(); err == nil {
				reinitIPv6Addr = ipv6.Addr
				filterMgr.Setup6(ipv6.Addr, ipv6.PrefixLen)
			}

			self.mu.Lock()
			self.tcFilterMgr = filterMgr
			self.mu.Unlock()
		}

		if reinitIPv6Addr == "" {
			if ipv6, err := NewNetworkInterface(self.name).IpV6Addr(); err == nil {
				reinitIPv6Addr = ipv6.Addr
			}
		}
		err = nftables.SetupCaptivePortal(i.Device, ipv4.Addr, reinitIPv6Addr)
		if err != nil {
			return nil, err
		}
		if derr := captivedns.Setup(ipv4.Addr); derr != nil {
		}
		return nil, nil
	})

	return err
}

func (self *NetworkLan) SetupCaptivePortal() (err error) {
	_, err = networkQue.Exec("SetupCaptivePortal", func() (interface{}, error) {
		iface := self.GetInterface()
		info, err := iface.getInfo()
		if err != nil {
			return nil, err
		}
		ipv4, err := iface.IpV4Addr()
		if err != nil {
			return nil, err
		}
		// IPv6 router address is optional — captive portal still works with IPv4 only
		routerIp6 := ""
		if ipv6, err := iface.IpV6Addr(); err == nil {
			routerIp6 = ipv6.Addr
		}
		if err = nftables.SetupCaptivePortal(info.Device, ipv4.Addr, routerIp6); err != nil {
			return nil, err
		}
		captivedns.Setup(ipv4.Addr)
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

		if c, ok := cfg.Lans[self.name]; ok {
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
			// Fetch once and reuse for the captive portal call below.
			routerIp6setup := ""
			if ipv6, err := NewNetworkInterface(self.name).IpV6Addr(); err == nil {
				routerIp6setup = ipv6.Addr
				if err := filterMgr.Setup6(ipv6.Addr, ipv6.PrefixLen); err != nil {
				}
			}

			self.mu.Lock()
			self.tcFilterMgr = filterMgr
			self.mu.Unlock()
			err = nftables.SetupCaptivePortal(i.Device, ipv4.Addr, routerIp6setup)
			if err != nil {
				return nil, err
			}
			captivedns.Setup(ipv4.Addr)

			return nil, nil

		}

		return nil, errors.New(self.name + "network config not found or traffic shaping not enabled")
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
		return nil, self.tcClassMgr.DeleteClass(tc.TcClassId(classid))
	})
	return err
}

func (self *NetworkLan) CreateFilter(ip string, classid uint) error {
	_, err := networkQue.Exec("CreateFilter", func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()
		return nil, self.tcFilterMgr.CreateFilter(ip, tc.TcClassId(classid))
	})
	return err
}

func (self *NetworkLan) DelFilter(ip string, classid uint) error {
	_, err := networkQue.Exec("DelFilter", func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()
		return nil, self.tcFilterMgr.DeleteFilter(ip)
	})
	return err
}

func (self *NetworkLan) UpdateBandwidth(downMbits int, upMbits int) error {
	_, err := networkQue.Exec("UpdateBandwidth", func() (any, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()

		downKbit := tc.Mbit(downMbits).ToKbit()
		upKbit := tc.Mbit(upMbits).ToKbit()
		return nil, self.tcClassMgr.UpdateBandwidth(downKbit, upKbit)
	})
	return err
}
