package tc

import (
	"errors"
	"fmt"
	"log"
	"sync"

	ifbutil "core/internal/modules/network"
	jobque "core/tools/job-que"
	cmd "core/tools/shell"
)

var (
	filterQue = jobque.NewJobQue[any]()
)

type TcIpField string

const (
	TcIpFieldSrc TcIpField = "src"
	TcIpFieldDst TcIpField = "dst"
)

type TcFilter struct {
	dev        string
	ipsegmt    *ipsegmt
	filterList map[string]string
	mu         sync.RWMutex
}

func NewTcFilter(dev string, ip string, netmask int) (*TcFilter, error) {
	if netmask < 17 {
		return nil, errors.New("Minimum network mask is 17")
	}

	seg, err := newIpsegmt(ip, netmask)
	if err != nil {
		log.Println("tc error: " + err.Error())
		return nil, err
	}

	flist := map[string]string{}
	return &TcFilter{
		dev:        dev,
		ipsegmt:    seg,
		filterList: flist,
	}, nil
}

func (self *TcFilter) devs() []string {
	if ifbutil.IsIfbSupported() {
		return []string{self.dev, ifbName(self.dev)}
	}
	return []string{self.dev}
}

func (self *TcFilter) tcpField(dev string) TcIpField {
	if dev == self.dev {
		return TcIpFieldDst
	}
	return TcIpFieldSrc
}

func (self *TcFilter) maskPosition(dev string) uint8 {
	field := self.tcpField(dev)
	if field == TcIpFieldSrc {
		return 12
	}
	if field == TcIpFieldDst {
		return 16
	}
	return 0
}

// Return hash bucket handle for a given ip
// TODO: Fix the max netmask limitation of /17
func (self *TcFilter) hashBktFor(clientIp string) (hex string, err error) {
	ipsg, err := newIpsegmt(clientIp, self.ipsegmt.netmask)
	if err != nil {
		return hex, err
	}

	var ht int
	if ipsg.netmask < 24 {
		ht = ipsg.segVal(2) + 2
	} else {
		ht = 1
	}
	bkt := ipsg.segVal(3)
	return fmt.Sprintf("%x:%x:800", ht, bkt), nil
}

func (self *TcFilter) addFilterList(ip string, classid string) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.filterList[ip] = classid
}

func (self *TcFilter) removeFilterList(ip string) {
	self.mu.Lock()
	defer self.mu.Unlock()
	delete(self.filterList, ip)
}

func (self *TcFilter) create(clientIp string, classid string) (err error) {
	htBkt, err := self.hashBktFor(clientIp)
	if err != nil {
		return err
	}
	for _, dev := range self.devs() {
		err = cmd.Exec(fmt.Sprintf("tc filter add dev %s parent 1:0 protocol ip prio 10 u32 ht %s match ip %s %s flowid %s", dev, htBkt, self.tcpField(dev), clientIp, classid), nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *TcFilter) delete(clientIp string) (err error) {
	htBkt, err := self.hashBktFor(clientIp)
	if err != nil {
		return err
	}
	for _, dev := range self.devs() {
		err = cmd.Exec(fmt.Sprintf("tc filter del dev %s parent 1:0 handle %s prio 10 u32", dev, htBkt), nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *TcFilter) Setup() error {
	_, err := filterQue.Exec(func() (any, error) {
		for _, dev := range self.devs() {
			cmds := []string{}
			count := len(self.ipsegmt.segments)
			ht := 1

			// clean up old filters (ignore errors - filters may not exist on first setup)
			cmd.Exec(fmt.Sprintf("tc filter del dev %s parent 1:0 prio 10 protocol ip u32", dev), nil)
			cmd.Exec(fmt.Sprintf("tc filter del dev %s parent 1:0 prio 100 protocol ip u32", dev), nil)

			for segIndex := 0; segIndex < count; segIndex++ {
				if self.ipsegmt.hostMasked(segIndex) {
					divisor := self.ipsegmt.segMaxVal(segIndex) + 1
					pos := self.maskPosition(dev)
					mask := self.ipsegmt.segMaskHex(segIndex)
					netip := self.ipsegmt.baseIp()
					netmask := self.ipsegmt.netmask
					f := self.tcpField(dev)

					if ht == 1 {
						cmds = append(cmds, []string{
							fmt.Sprintf("tc filter add dev %s parent 1:0 prio 10 protocol ip u32", dev),
							fmt.Sprintf("tc filter add dev %s parent 1:0 protocol ip prio 10 handle %x: u32 divisor %d", dev, ht, divisor),
							fmt.Sprintf("tc filter add dev %s parent 1:0 protocol ip prio 100 u32 ht 800:: match ip %s %s/%d hashkey mask %s at %d link %x:", dev, f, netip, netmask, mask, pos, ht),
						}...)
						ht += 1
					} else {
						parentSegIndex := segIndex - 1
						if self.ipsegmt.hostMasked(parentSegIndex) {
							parentHt := ht - 1
							listIndex := self.ipsegmt.segMinVal(parentSegIndex)
							maxIndex := self.ipsegmt.segMaxVal(parentSegIndex)
							for i := listIndex; i <= maxIndex; i++ {
								cmds = append(cmds, []string{
									fmt.Sprintf("tc filter add dev %s parent 1:0 protocol ip prio 10 handle %x: u32 divisor %d", dev, ht, 256),
									fmt.Sprintf("tc filter add dev %s parent 1:0 protocol ip prio 10 u32 ht %x:%x: match ip %s %s/%d hashkey mask %s at %d link %x:", dev, parentHt, i, self.tcpField(dev), netip, netmask, mask, pos, ht),
								}...)
								ht += 1
							}
						}
					}
				}
			}

			for _, c := range cmds {
				err := cmd.Exec(c, nil)
				if err != nil {
					log.Println("Error in tc filter setup: ", err)
					return nil, err
				}
			}
		}

		return nil, nil
	})

	return err
}

func (self *TcFilter) Reset() (err error) {
	err = self.Setup()
	if err != nil {
		return err
	}

	_, err = filterQue.Exec(func() (any, error) {
		self.mu.RLock()
		filterListCopy := make(map[string]string, len(self.filterList))
		for ip, classid := range self.filterList {
			filterListCopy[ip] = classid
		}
		self.mu.RUnlock()

		for ip, classid := range filterListCopy {
			if err := self.create(ip, classid); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

	return err
}

// Create a tc filter for client ip and classid
func (self *TcFilter) CreateFilter(clientIp string, classid string) error {
	_, err := filterQue.Exec(func() (any, error) {
		err := self.create(clientIp, classid)
		if err != nil {
			return nil, err
		}
		self.addFilterList(clientIp, classid)
		return nil, nil
	})
	return err
}

// Delete a tc filter
func (self *TcFilter) DeleteFilter(clientIp string) error {
	_, err := filterQue.Exec(func() (any, error) {
		err := self.delete(clientIp)
		if err != nil {
			return nil, err
		}
		self.removeFilterList(clientIp)
		return nil, nil
	})
	return err
}

func (self *TcFilter) CleanUp() error {
	_, err := filterQue.Exec(func() (any, error) {
		// Ignore errors during cleanup - filters may not exist yet
		cmd.Exec(fmt.Sprintf("tc filter del dev %s parent 1:0 prio 10 protocol ip u32", self.dev), nil)

		if ifbutil.IsIfbSupported() {
			ifb := ifbName(self.dev)
			cmd.Exec(fmt.Sprintf("tc filter del dev %s parent 1:0 prio 10 protocol ip u32", ifb), nil)
		}

		return nil, nil
	})

	return err
}
