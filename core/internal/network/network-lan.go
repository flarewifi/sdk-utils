package network

import (
	"errors"
	"log"
	"sync"

	"core/internal/modules/nftables"
	"core/internal/modules/tc"
	"core/internal/modules/ubus"
	"core/utils/config"
	jobque "core/utils/job-que"
)

var (
	networkQue = jobque.NewJobQue[any]()
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
	_, err = networkQue.Exec(func() (any, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()

		if self.tcClassMgr == nil || self.tcFilterMgr == nil {
			log.Printf("WARNING: TC managers not initialized for LAN '%s', skipping reset", self.name)
			return nil, errors.New("TC managers not initialized")
		}

		err = self.tcClassMgr.Reset()
		if err != nil {
			log.Println(err)
			return nil, err
		}

		err = self.tcFilterMgr.Reset()
		if err != nil {
			log.Println(err)
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
	_, err = networkQue.Exec(func() (any, error) {
		log.Printf("Reinitializing TC for LAN '%s'...", self.name)

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
			log.Printf("ERROR: Failed to read bandwidth config for LAN '%s': %v", self.name, err)
			return nil, err
		}

		i, err := ubus.GetNetworkInterface(self.name)
		if err != nil {
			log.Printf("ERROR: Failed to get interface info for LAN '%s': %v", self.name, err)
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
				log.Printf("Auto-detected link speed for device '%s': %d Mbps (raw: %s)", dev, detectedSpeed, netDev.Speed)
			} else {
				log.Printf("WARNING: Could not detect link speed for device '%s', using default: %d Mbps", dev, defaultSpeed)
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
			log.Printf("ERROR: Failed to get IPv4 address for LAN '%s': %v", self.name, err)
			return nil, err
		}

		// If TC managers exist, use Reset() to preserve session data
		// Otherwise, create new managers
		if oldClassMgr != nil && oldFilterMgr != nil {
			log.Printf("Resetting existing TC managers for LAN '%s' (preserving active sessions)", self.name)

			// Reset TC Class Manager (preserves classList)
			log.Printf("Resetting TC classes for LAN '%s' on device '%s' (down: %d Mbps, up: %d Mbps)",
				self.name, dev, lanCfg.GlobalDownMbits, lanCfg.GlobalUpMbits)
			err = oldClassMgr.Reset()
			if err != nil {
				log.Printf("ERROR: TcClassMgr Reset() failed for LAN '%s': %v", self.name, err)
				return nil, err
			}

			// Reset TC Filter Manager (preserves filterList)
			log.Printf("Resetting TC filters for LAN '%s' with IP %s/%d", self.name, ipv4.Addr, ipv4.Netmask)
			err = oldFilterMgr.Reset()
			if err != nil {
				log.Printf("ERROR: TcFilterMgr Reset() failed for LAN '%s': %v", self.name, err)
				return nil, err
			}

			log.Printf("Successfully preserved and recreated TC rules for all active sessions on LAN '%s'", self.name)
		} else {
			// First time setup or managers were nil
			log.Printf("Creating new TC managers for LAN '%s' (no existing sessions to preserve)", self.name)

			// Setup TC Class Manager
			log.Printf("Setting up TC classes for LAN '%s' on device '%s' (down: %d Mbps, up: %d Mbps)",
				self.name, dev, lanCfg.GlobalDownMbits, lanCfg.GlobalUpMbits)
			classMgr := tc.NewTcClassMgr(dev, tc.Kbit(lanCfg.GlobalDownMbits*1000), tc.Kbit(lanCfg.GlobalUpMbits*1000))
			err = classMgr.Setup()
			if err != nil {
				log.Printf("ERROR: TcClassMgr Setup() failed for LAN '%s': %v", self.name, err)
				return nil, err
			}

			self.mu.Lock()
			self.tcClassMgr = classMgr
			self.mu.Unlock()

			// Setup TC Filter Manager
			log.Printf("Setting up TC filters for LAN '%s' with IP %s/%d", self.name, ipv4.Addr, ipv4.Netmask)
			filterMgr := tc.NewTcFilterMgr(i.Device)
			err = filterMgr.Setup(ipv4.Addr, ipv4.Netmask)
			if err != nil {
				log.Printf("ERROR: TcFilterMgr Setup() failed for LAN '%s': %v", self.name, err)
				return nil, err
			}

			self.mu.Lock()
			self.tcFilterMgr = filterMgr
			self.mu.Unlock()
		}

		// Setup Captive Portal
		log.Printf("Setting up captive portal for LAN '%s' on device '%s' with IP %s", self.name, i.Device, ipv4.Addr)
		err = nftables.SetupCaptivePortal(i.Device, ipv4.Addr)
		if err != nil {
			log.Printf("ERROR: Captive portal setup failed for LAN '%s': %v", self.name, err)
			return nil, err
		}

		log.Printf("TC reinitialization complete for LAN '%s'", self.name)
		return nil, nil
	})

	return err
}

func (self *NetworkLan) SetupCaptivePortal() (err error) {
	_, err = networkQue.Exec(func() (interface{}, error) {
		iface := self.GetInterface()
		info, err := iface.getInfo()
		if err != nil {
			return nil, err
		}
		ipv4, err := iface.IpV4Addr()
		if err != nil {
			return nil, err
		}
		err = nftables.SetupCaptivePortal(info.Device, ipv4.Addr)
		return nil, err
	})

	return err
}

func (self *NetworkLan) SetupTrafficControl() (err error) {
	_, err = networkQue.Exec(func() (interface{}, error) {
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
					log.Printf("Auto-detected link speed for device '%s': %d Mbps (raw: %s)", dev, detectedSpeed, netDev.Speed)
				} else {
					log.Printf("WARNING: Could not detect link speed for device '%s', using default: %d Mbps", dev, defaultSpeed)
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
				log.Println("TcClassMgr Setup() Error: ", err)
				return nil, err
			}

			self.mu.Lock()
			self.tcClassMgr = classMgr
			self.mu.Unlock()

			ipv4, err := i.IpV4Addr()
			if err != nil {
				log.Println("TcFilterMgr Setup() Error: ", err)
				return nil, err
			}

			filterMgr := tc.NewTcFilterMgr(i.Device)
			err = filterMgr.Setup(ipv4.Addr, ipv4.Netmask)
			if err != nil {
				return nil, err
			}

			self.mu.Lock()
			self.tcFilterMgr = filterMgr
			self.mu.Unlock()

			err = nftables.SetupCaptivePortal(i.Device, ipv4.Addr)
			if err != nil {
				return nil, err
			}

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
	_, err := networkQue.Exec(func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()

		downKbit := tc.Kbit(downMbit * 1000)
		upKbit := tc.Kbit(upMbit * 1000)

		return nil, self.tcClassMgr.CreateClass(self.tcClassMgr.UserTcClass(), tc.TcClassId(classid), 1, 1, downKbit, upKbit)
	})
	return err
}

func (self *NetworkLan) ChangeClass(classid uint, downMbit int, upMbit int) error {
	_, err := networkQue.Exec(func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()

		downKbit := tc.Kbit(downMbit * 1000)
		upKbit := tc.Kbit(upMbit * 1000)

		return nil, self.tcClassMgr.ChangeClass(self.tcClassMgr.UserTcClass(), tc.TcClassId(classid), 1, 1, downKbit, upKbit)
	})
	return err
}

func (self *NetworkLan) DelClass(classid uint) error {
	_, err := networkQue.Exec(func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()
		return nil, self.tcClassMgr.DeleteClass(tc.TcClassId(classid))
	})
	return err
}

func (self *NetworkLan) CreateFilter(ip string, classid uint) error {
	_, err := networkQue.Exec(func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()
		return nil, self.tcFilterMgr.CreateFilter(ip, tc.TcClassId(classid))
	})
	return err
}

func (self *NetworkLan) DelFilter(ip string, classid uint) error {
	_, err := networkQue.Exec(func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()
		return nil, self.tcFilterMgr.DeleteFilter(ip)
	})
	return err
}

func (self *NetworkLan) UpdateBandwidth(downMbits int, upMbits int) error {
	_, err := networkQue.Exec(func() (any, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()

		downKbit := tc.Mbit(downMbits).ToKbit()
		upKbit := tc.Mbit(upMbits).ToKbit()
		return nil, self.tcClassMgr.UpdateBandwidth(downKbit, upKbit)
	})
	return err
}
