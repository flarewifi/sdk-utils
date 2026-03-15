package tc

import (
	"fmt"
	"log"
	"sync"

	ifbutil "core/internal/modules/network"
	cmd "core/utils/shell"
)

// TcFilter6 manages tc u32 filters for IPv6 clients on a single LAN device.
// It mirrors the TcFilter (IPv4) API but uses "protocol ipv6" and "match ip6"
// tc commands, and operates on 128-bit IPv6 addresses.
//
// Hash table strategy:
//   - We use a single two-level hash: the first level hashes on bytes in the
//     host portion of the prefix, the second hashes on the remaining bytes.
//   - For most /64 subnets (typical IPv6 LAN allocation) we bucket on bytes
//     12-15 of the address (the last 32 host bits), giving 256 buckets of 256
//     each — identical bucket granularity to the IPv4 /24 case.
//   - For other prefix lengths we select the two most-significant host bytes
//     for the two-level hash, falling back to a flat filter if the prefix is
//     longer than /120 (fewer than 256 possible hosts).
type TcFilter6 struct {
	dev        string
	ipsegmt    *ipsegmt6
	filterList map[string]string
	mu         sync.RWMutex
}

func NewTcFilter6(dev string, ip string, prefixLen int) (*TcFilter6, error) {
	seg, err := newIpsegmt6(ip, prefixLen)
	if err != nil {
		log.Println("tc6 error: " + err.Error())
		return nil, err
	}

	return &TcFilter6{
		dev:        dev,
		ipsegmt:    seg,
		filterList: make(map[string]string),
	}, nil
}

// devs returns the list of devices that filters must be applied to.
// When IFB is supported, both the LAN device (ingress→egress redirect) and the
// corresponding IFB device (for upload shaping) are returned.
func (self *TcFilter6) devs() []string {
	if ifbutil.IsIfbSupported() {
		return []string{self.dev, ifbName(self.dev)}
	}
	return []string{self.dev}
}

// tcpField6 returns "dst" for the LAN device (download path) and "src" for
// the IFB device (upload path, traffic coming from the client).
func (self *TcFilter6) tcpField6(dev string) TcIpField {
	if dev == self.dev {
		return TcIpFieldDst
	}
	return TcIpFieldSrc
}

// hashBktFor6 computes the tc hash-bucket handle string for a client IPv6 address.
//
// Setup() assigns level-2 hash-table handles sequentially starting at htLevel2=2,
// one per first-host-byte bucket, iterating from minBkt to maxBkt:
//
//	bucket minBkt   → handle 2
//	bucket minBkt+1 → handle 3
//	...
//	bucket i        → handle 2 + (i - minBkt)
//
// hashBktFor6 must use the same formula so that CreateFilter/DeleteFilter target
// the exact handles that Setup() created.  minBkt is derived from the network
// address (self.ipsegmt), not from the client IP.
func (self *TcFilter6) hashBktFor6(clientIp string) (string, error) {
	clientSeg, err := newIpsegmt6(clientIp, self.ipsegmt.prefixLen)
	if err != nil {
		return "", fmt.Errorf("hashBktFor6: %w", err)
	}

	// Find the first two host-masked byte indices (same logic as Setup)
	firstHostByte := -1
	secondHostByte := -1
	for i := 0; i < 16; i++ {
		if clientSeg.hostMasked6(i) {
			if firstHostByte < 0 {
				firstHostByte = i
			} else if secondHostByte < 0 {
				secondHostByte = i
				break
			}
		}
	}

	if firstHostByte < 0 {
		// /128 prefix — only one possible host, target the flat root bucket
		return "800::800", nil
	}

	// Compute the level-2 handle for this client's first-host-byte value.
	// minBkt is the minimum value of that byte in the network's address range.
	minBkt := self.ipsegmt.segMinVal6(firstHostByte)
	clientFirstByteVal := clientSeg.segVal6(firstHostByte)
	ht := 2 + (clientFirstByteVal - minBkt)

	if secondHostByte < 0 {
		// Single-level hash — bucket 0 within the level-1 table
		return fmt.Sprintf("%x:0:800", ht), nil
	}

	// Second level: bucket is the client's value of the second host byte.
	// Setup creates one entry per value (minBkt2..maxBkt2), so the handle
	// is the byte value itself (tc uses the hash key directly as bucket index).
	bkt := clientSeg.segVal6(secondHostByte)
	return fmt.Sprintf("%x:%x:800", ht, bkt), nil
}

func (self *TcFilter6) addFilterList(ip string, classid string) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.filterList[ip] = classid
}

func (self *TcFilter6) removeFilterList(ip string) {
	self.mu.Lock()
	defer self.mu.Unlock()
	delete(self.filterList, ip)
}

func (self *TcFilter6) create6(clientIp string, classid string) error {
	htBkt, err := self.hashBktFor6(clientIp)
	if err != nil {
		return err
	}
	for _, dev := range self.devs() {
		field := self.tcpField6(dev)
		err = cmd.Exec(fmt.Sprintf(
			"tc filter add dev %s parent 1:0 protocol ipv6 prio 10 u32 ht %s match ip6 %s %s flowid %s",
			dev, htBkt, field, clientIp, classid,
		), nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *TcFilter6) delete6(clientIp string) error {
	htBkt, err := self.hashBktFor6(clientIp)
	if err != nil {
		return err
	}
	for _, dev := range self.devs() {
		err = cmd.Exec(fmt.Sprintf(
			"tc filter del dev %s parent 1:0 handle %s prio 10 u32",
			dev, htBkt,
		), nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// Setup initialises the tc hash tables for IPv6 on each device.
// It deletes any existing IPv6 filters first to start clean.
func (self *TcFilter6) Setup() error {
	_, err := filterQue.Exec("TcFilter6.Setup", func() (any, error) {
		for _, dev := range self.devs() {
			// Clean up old IPv6 filters (ignore errors — may not exist on first setup)
			cmd.Exec(fmt.Sprintf("tc filter del dev %s parent 1:0 prio 10 protocol ipv6 u32", dev), nil)
			cmd.Exec(fmt.Sprintf("tc filter del dev %s parent 1:0 prio 100 protocol ipv6 u32", dev), nil)

			// Build the two-level hash tables for the host portion of the prefix.
			// Find first and second host-masked byte indices.
			firstHostByte := -1
			secondHostByte := -1
			for i := 0; i < 16; i++ {
				if self.ipsegmt.hostMasked6(i) {
					if firstHostByte < 0 {
						firstHostByte = i
					} else if secondHostByte < 0 {
						secondHostByte = i
						break
					}
				}
			}

			if firstHostByte < 0 {
				// /128 — no hash needed
				continue
			}

			field := self.tcpField6(dev)
			netip := self.ipsegmt.baseIp6()
			prefixLen := self.ipsegmt.prefixLen

			htLevel1 := 1
			divisor1 := self.ipsegmt.segMaxVal6(firstHostByte) - self.ipsegmt.segMinVal6(firstHostByte) + 1
			mask1 := self.ipsegmt.segMaskHex6(firstHostByte)
			pos1 := maskPosition6(firstHostByte, field)

			cmds := []string{
				// Enable u32 filtering on this device
				fmt.Sprintf("tc filter add dev %s parent 1:0 prio 10 protocol ipv6 u32", dev),
				// Create the first-level hash table
				fmt.Sprintf("tc filter add dev %s parent 1:0 protocol ipv6 prio 10 handle %x: u32 divisor %d", dev, htLevel1, divisor1),
				// Root entry: hash on the first host byte → link to htLevel1
				fmt.Sprintf("tc filter add dev %s parent 1:0 protocol ipv6 prio 100 u32 ht 800:: match ip6 %s %s/%d hashkey mask %s at %d link %x:",
					dev, field, netip, prefixLen, mask1, pos1, htLevel1),
			}

			if secondHostByte >= 0 {
				// Two-level hash: enumerate all first-level buckets and link to second-level tables
				minBkt := self.ipsegmt.segMinVal6(firstHostByte)
				maxBkt := self.ipsegmt.segMaxVal6(firstHostByte)
				divisor2 := self.ipsegmt.segMaxVal6(secondHostByte) - self.ipsegmt.segMinVal6(secondHostByte) + 1
				mask2 := self.ipsegmt.segMaskHex6(secondHostByte)
				pos2 := maskPosition6(secondHostByte, field)

				htLevel2 := htLevel1 + 1
				for i := minBkt; i <= maxBkt; i++ {
					cmds = append(cmds,
						fmt.Sprintf("tc filter add dev %s parent 1:0 protocol ipv6 prio 10 handle %x: u32 divisor %d", dev, htLevel2, divisor2),
						fmt.Sprintf("tc filter add dev %s parent 1:0 protocol ipv6 prio 10 u32 ht %x:%x: match ip6 %s %s/%d hashkey mask %s at %d link %x:",
							dev, htLevel1, i, field, netip, prefixLen, mask2, pos2, htLevel2),
					)
					htLevel2++
				}
			}

			for _, c := range cmds {
				if err := cmd.Exec(c, nil); err != nil {
					log.Println("Error in tc6 filter setup: ", err)
					return nil, err
				}
			}
		}
		return nil, nil
	})
	return err
}

// Reset re-runs Setup and then recreates all filters from the cached list.
func (self *TcFilter6) Reset() error {
	if err := self.Setup(); err != nil {
		return err
	}

	_, err := filterQue.Exec("TcFilter6.Reset.restoreFilters", func() (any, error) {
		self.mu.RLock()
		filterListCopy := make(map[string]string, len(self.filterList))
		for ip, classid := range self.filterList {
			filterListCopy[ip] = classid
		}
		self.mu.RUnlock()

		for ip, classid := range filterListCopy {
			if err := self.create6(ip, classid); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}

// CreateFilter adds a tc filter for a client IPv6 address and the given HTB class ID.
func (self *TcFilter6) CreateFilter(clientIp string, classid string) error {
	_, err := filterQue.Exec("TcFilter6.CreateFilter", func() (any, error) {
		if err := self.create6(clientIp, classid); err != nil {
			return nil, err
		}
		self.addFilterList(clientIp, classid)
		return nil, nil
	})
	return err
}

// DeleteFilter removes the tc filter for a client IPv6 address.
func (self *TcFilter6) DeleteFilter(clientIp string) error {
	_, err := filterQue.Exec("TcFilter6.DeleteFilter", func() (any, error) {
		if err := self.delete6(clientIp); err != nil {
			return nil, err
		}
		self.removeFilterList(clientIp)
		return nil, nil
	})
	return err
}

// CleanUp removes all IPv6 u32 filters from the device (and its IFB peer).
func (self *TcFilter6) CleanUp() error {
	_, err := filterQue.Exec("TcFilter6.CleanUp", func() (any, error) {
		cmd.Exec(fmt.Sprintf("tc filter del dev %s parent 1:0 prio 10 protocol ipv6 u32", self.dev), nil)
		if ifbutil.IsIfbSupported() {
			ifb := ifbName(self.dev)
			cmd.Exec(fmt.Sprintf("tc filter del dev %s parent 1:0 prio 10 protocol ipv6 u32", ifb), nil)
		}
		return nil, nil
	})
	return err
}
