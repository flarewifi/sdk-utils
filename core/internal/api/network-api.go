package api

import (
	cnet "core/internal/network"
	"core/internal/utils/ubus"
	sdkapi "sdk/api"
)

func NewNetworkApi(api *PluginApi, trfk *cnet.TrafficMgr) {
	networkApi := &NetworkApi{trfk}
	api.NetworkAPI = networkApi
}

type NetworkApi struct {
	trfk *cnet.TrafficMgr
}

func (self *NetworkApi) ListDevices() (netdevs []sdkapi.INetworkDevice, err error) {
	devices, err := ubus.GetNetworkDevices()
	if err != nil {
		return nil, err
	}

	netdevs = []sdkapi.INetworkDevice{}
	for _, v := range devices {
		dev := cnet.NewNetworkDevice(v)
		netdevs = append(netdevs, dev)
	}

	return netdevs, nil
}

func (self *NetworkApi) ListInterfaces() (interfaces []sdkapi.INetworkInterface, err error) {
	ifaces, err := ubus.GetNetworkInterfaces()
	if err != nil {
		return nil, err
	}

	for ifname := range ifaces {
		iface := cnet.NewNetworkInterface(ifname)
		interfaces = append(interfaces, iface)
	}

	return interfaces, nil
}

func (self *NetworkApi) GetDevice(name string) (sdkapi.INetworkDevice, error) {
	dev, err := ubus.GetNetworkDevice(name)
	if err != nil {
		return nil, err
	}
	return cnet.NewNetworkDevice(dev), nil
}

func (self *NetworkApi) GetInterface(name string) (sdkapi.INetworkInterface, error) {
	_, err := ubus.GetNetworkInterface(name)
	if err != nil {
		return nil, err
	}
	return cnet.NewNetworkInterface(name), nil
}

func (self *NetworkApi) FindByIp(clientIp string) (sdkapi.INetworkInterface, error) {
	iface, err := cnet.FindByIp(clientIp)
	if err != nil {
		return nil, err
	}

	return cnet.NewNetworkInterface(iface.Name()), nil
}

func (self *NetworkApi) Traffic() sdkapi.ITrafficApi {
	return self.trfk
}
