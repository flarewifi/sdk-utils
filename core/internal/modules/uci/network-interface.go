package uci

import (
	"errors"
	"net"

	sdkapi "sdk/api"
)

// interface
func (self *UciNetworkApi) GetInterface(section string) (iface *sdkapi.INetIface, err error) {
	var ifdata sdkapi.INetIface

	ifdata.Section = section

	devices, ok := UciTree.Get("network", section, "device")
	if ok && len(devices) > 0 {
		ifdata.Device = devices[0]
	}

	protos, ok := UciTree.Get("network", section, "proto")
	if ok && len(protos) > 0 {
		ifdata.Proto = protos[0]
	} else {
		return nil, errors.New("Can't get protocol value of " + section)
	}

	if ifdata.Proto == "static" {
		addrs, ok := UciTree.Get("network", section, "ipaddr")
		if !ok || len(addrs) == 0 {
			return nil, errors.New("Can't get ipaddr value of " + section)
		}

		// Modern UCI configs embed the prefix length directly in ipaddr (e.g.
		// `list ipaddr '10.0.0.1/20'`) and omit a separate "netmask" option
		// entirely; older configs keep ipaddr and netmask as separate options.
		if ip, ipnet, cidrErr := net.ParseCIDR(addrs[0]); cidrErr == nil {
			ifdata.IpAddr = ip.String()
			ifdata.Netmask = net.IP(ipnet.Mask).String()
		} else {
			ifdata.IpAddr = addrs[0]

			netmasks, ok := UciTree.Get("network", section, "netmask")
			if ok && len(netmasks) > 0 {
				ifdata.Netmask = netmasks[0]
			} else {
				return nil, errors.New("Can't get netmask value of " + section)
			}
		}

		gateways, ok := UciTree.Get("network", section, "gateway")
		if ok && len(gateways) > 0 {
			ifdata.Gateway = gateways[0]
		}
	}

	return &ifdata, nil

}

func (self *UciNetworkApi) GetInterfaceSecs() (sections []string) {
	secs, _ := UciTree.GetSections("network", "interface")
	return secs
}

func (self *UciNetworkApi) GetInterfaces() (ifaces []*sdkapi.INetIface, err error) {
	secs, _ := UciTree.GetSections("network", "interface")

	for _, s := range secs {
		// Skip a single malformed/partial section instead of discarding every
		// other interface that parsed fine.
		if iface, ferr := self.GetInterface(s); ferr == nil {
			ifaces = append(ifaces, iface)
		}
	}

	return ifaces, nil
}

func (self *UciNetworkApi) SetInterface(section string, cfg *sdkapi.INetIface) error {
	var ok bool

	ok = UciTree.Set("network", section, "proto", cfg.Proto)
	if !ok {
		return errors.New("Can't set proto value of " + section)
	}

	ok = UciTree.Set("network", section, "device", cfg.Device)
	if !ok {
		return errors.New("Can't set device value of " + section)
	}

	if cfg.Proto == "static" {
		ok = UciTree.Set("network", section, "ipaddr", cfg.IpAddr)
		if !ok {
			return errors.New("Can't set ipaddr value of " + section)
		}

		ok = UciTree.Set("network", section, "netmask", cfg.Netmask)
		if !ok {
			return errors.New("Can't set netmask value of " + section)
		}

		if cfg.Gateway != "" {
			ok = UciTree.Set("network", section, "gateway", cfg.Gateway)
			if !ok {
				return errors.New("Can't set gateway value of " + section)
			}
		}
	} else {
		UciTree.Del("network", section, "ipaddr")
		UciTree.Del("network", section, "netmask")
		UciTree.Del("network", section, "gateway")
	}

	return nil

}
