package network

import (
	"errors"
	"fmt"
	"log"
	"net"

	"core/internal/utils/ubus"
	sdkapi "sdk/api"
)

type NetworkInterface struct {
	name string
}

func (self *NetworkInterface) getInfo() (*ubus.NetworkInterface, error) {
	return ubus.GetNetworkInterface(self.name)
}

func (self *NetworkInterface) Ifname() string {
	return self.name
}

func (self *NetworkInterface) Device() (d sdkapi.INetworkDevice, err error) {
	info, err := self.getInfo()
	if err != nil {
		return nil, err
	}
	dev, err := ubus.GetNetworkDevice(info.Device)
	if err != nil {
		return nil, err
	}
	d = NewNetworkDevice(dev)
	return d, nil
}

func (self *NetworkInterface) IpV4Addr() (*sdkapi.NetworkIpv4, error) {
	info, err := self.getInfo()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if len(info.IpV4Addresses) > 0 {
		addr := info.IpV4Addresses[0]
		return &sdkapi.NetworkIpv4{
			Addr:    addr.Addr,
			Netmask: addr.Netmask,
		}, nil
	}

	return nil, errors.New("Cannot determine network interface IP.")
}

func (self *NetworkInterface) IPNet() (*net.IPNet, error) {
	ipv4, err := self.IpV4Addr()
	if err != nil {
		return nil, err
	}

	_, ipnet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", ipv4.Addr, ipv4.Netmask))
	if err != nil {
		return nil, err
	}

	return ipnet, err
}

func (self *NetworkInterface) Up() bool {
	info, err := self.getInfo()
	if err != nil {
		return false
	}
	return info.Up
}

func NewNetworkInterface(name string) *NetworkInterface {
	return &NetworkInterface{name}
}
