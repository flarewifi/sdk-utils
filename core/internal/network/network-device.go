package network

import (
	"strings"
	"sync"
	"time"

	"core/internal/modules/ubus"
	sdkapi "sdk/api"
)

// deviceRateCache tracks previous readings for rate calculation per device
var deviceRateCache = struct {
	sync.Mutex
	data map[string]*deviceRateData
}{
	data: make(map[string]*deviceRateData),
}

type deviceRateData struct {
	timestamp time.Time
	rxBytes   uint
	txBytes   uint
	rxRate    uint64
	txRate    uint64
}

type NetworkDevice struct {
	netdev *ubus.NetworkDevice
}

func (self *NetworkDevice) Name() string {
	return self.netdev.Name
}

func (self *NetworkDevice) Type() sdkapi.NetDevType {
	return sdkapi.NetDevType(self.netdev.Type)
}

func (self *NetworkDevice) MacAddr() string {
	return self.netdev.MacAddr
}

func (self *NetworkDevice) Up() bool {
	return self.netdev.Up
}

func (self *NetworkDevice) Carrier() bool {
	return self.netdev.Carrier
}

func (self *NetworkDevice) SpeedMbps() int {
	return ParseLinkSpeed(self.netdev.Speed)
}

func (self *NetworkDevice) Duplex() string {
	if self.netdev.Duplex == "" {
		return "unknown"
	}
	return self.netdev.Duplex
}

func (self *NetworkDevice) BridgeMembers() []string {
	return self.netdev.BridgeMembers
}

func (self *NetworkDevice) RxBytes() uint {
	return self.netdev.Stats.RxBytes
}

func (self *NetworkDevice) TxBytes() uint {
	return self.netdev.Stats.TxBytes
}

func (self *NetworkDevice) RxRate() uint64 {
	deviceRateCache.Lock()
	defer deviceRateCache.Unlock()

	now := time.Now()
	rxBytes := self.RxBytes()
	txBytes := self.TxBytes()
	name := self.Name()

	cached, exists := deviceRateCache.data[name]
	if !exists {
		// First call - initialize cache, return 0
		deviceRateCache.data[name] = &deviceRateData{
			timestamp: now,
			rxBytes:   rxBytes,
			txBytes:   txBytes,
		}
		return 0
	}

	elapsed := now.Sub(cached.timestamp).Seconds()
	if elapsed > 0 && rxBytes >= cached.rxBytes {
		cached.rxRate = uint64(float64(rxBytes-cached.rxBytes) / elapsed)
	}
	cached.timestamp = now
	cached.rxBytes = rxBytes
	cached.txBytes = txBytes

	return cached.rxRate
}

func (self *NetworkDevice) TxRate() uint64 {
	deviceRateCache.Lock()
	defer deviceRateCache.Unlock()

	now := time.Now()
	rxBytes := self.RxBytes()
	txBytes := self.TxBytes()
	name := self.Name()

	cached, exists := deviceRateCache.data[name]
	if !exists {
		// First call - initialize cache, return 0
		deviceRateCache.data[name] = &deviceRateData{
			timestamp: now,
			rxBytes:   rxBytes,
			txBytes:   txBytes,
		}
		return 0
	}

	elapsed := now.Sub(cached.timestamp).Seconds()
	if elapsed > 0 && txBytes >= cached.txBytes {
		cached.txRate = uint64(float64(txBytes-cached.txBytes) / elapsed)
	}
	cached.timestamp = now
	cached.rxBytes = rxBytes
	cached.txBytes = txBytes

	return cached.txRate
}

func (self *NetworkDevice) IsBridge() bool {
	return self.Type() == sdkapi.NetDevBridge
}

func (self *NetworkDevice) IsVlan() bool {
	return self.Type() == sdkapi.NetDevVLAN
}

func (self *NetworkDevice) IsIfb() bool {
	return strings.HasSuffix(self.Name(), "-ifb")
}

func (self *NetworkDevice) IsEthernet() bool {
	return self.Type() == sdkapi.NetDevEther
}

func (self *NetworkDevice) IsWireless() bool {
	return self.Type() == sdkapi.NetDevWLAN
}

func NewNetworkDevice(d *ubus.NetworkDevice) sdkapi.INetworkDevice {
	return &NetworkDevice{d}
}
