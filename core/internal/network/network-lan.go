package network

import (
	"errors"
	"log"
	"sync"

	"core/internal/config"
	jobque "core/internal/utils/job-que"
	"core/internal/utils/nftables"
	"core/internal/utils/tc"
	"core/internal/utils/ubus"
)

type NetworkLan struct {
	mu          sync.RWMutex
	queID       sync.Mutex
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
	_, err = jobque.Exec(&self.queID, func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()

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

func (self *NetworkLan) SetupCaptivePortal() (err error) {
	_, err = jobque.Exec(&self.queID, func() (interface{}, error) {
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

func (self *NetworkLan) SetupHFSC() (err error) {
	_, err = jobque.Exec(&self.queID, func() (interface{}, error) {
		cfg, err := config.ReadBandwidthConfig()
		if err != nil {
			return nil, err
		}

		i, err := ubus.GetNetworkInterface(self.name)
		if err != nil {
			return nil, err
		}

		if c, ok := cfg.Lans[self.name]; ok {
			if c.GlobalDownMbits == 0 {
				c.GlobalDownMbits = defaultSpeed
			}
			if c.GlobalUpMbits == 0 {
				c.GlobalUpMbits = defaultSpeed
			}

			dev := i.Device
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
	_, err := jobque.Exec(&self.queID, func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()

		downKbit := tc.Kbit(downMbit * 1000)
		upKbit := tc.Kbit(upMbit * 1000)

		return nil, self.tcClassMgr.CreateClass(self.tcClassMgr.UserTcClass(), tc.TcClassId(classid), 1, 1, downKbit, upKbit)
	})
	return err
}

func (self *NetworkLan) ChangeClass(classid uint, downMbit int, upMbit int) error {
	_, err := jobque.Exec(&self.queID, func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()

		downKbit := tc.Kbit(downMbit * 1000)
		upKbit := tc.Kbit(upMbit * 1000)

		return nil, self.tcClassMgr.ChangeClass(self.tcClassMgr.UserTcClass(), tc.TcClassId(classid), 1, 1, downKbit, upKbit)
	})
	return err
}

func (self *NetworkLan) DelClass(classid uint) error {
	_, err := jobque.Exec(&self.queID, func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()
		return nil, self.tcClassMgr.DeleteClass(tc.TcClassId(classid))
	})
	return err
}

func (self *NetworkLan) CreateFilter(ip string, classid uint) error {
	_, err := jobque.Exec(&self.queID, func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()
		return nil, self.tcFilterMgr.CreateFilter(ip, tc.TcClassId(classid))
	})
	return err
}

func (self *NetworkLan) DelFilter(ip string, classid uint) error {
	_, err := jobque.Exec(&self.queID, func() (interface{}, error) {
		self.mu.RLock()
		defer self.mu.RUnlock()
		return nil, self.tcFilterMgr.DeleteFilter(ip)
	})
	return err
}

func (self *NetworkLan) UpdateBandwidth(downMbits int, upMbits int) error {
	downKbit := tc.Mbit(downMbits).ToKbit()
	upKbit := tc.Mbit(upMbits).ToKbit()
	return self.tcClassMgr.UpdateBandwidth(downKbit, upKbit)
}
