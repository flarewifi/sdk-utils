package network

import (
	"sdk/api/network"
	"core/internal/utils/ubus"
)

type NetworkDevice struct {
	netdev *ubus.NetworkDevice
}

func (self *NetworkDevice) Name() string {
	return self.netdev.Name
}

func (self *NetworkDevice) Type() sdknet.NetDevType {
	return sdknet.NetDevType(self.netdev.Type)
}

func (self *NetworkDevice) MacAddr() string {
	return self.netdev.MacAddr
}

func (self *NetworkDevice) Up() bool {
	return self.netdev.Up
}

func (self *NetworkDevice) Speed() string {
	return self.Speed()
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

func NewNetworkDevice(d *ubus.NetworkDevice) sdknet.INetworkDevice {
	return &NetworkDevice{d}
}
