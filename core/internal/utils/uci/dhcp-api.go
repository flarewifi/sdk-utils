package uci

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	sdkapi "sdk/api"
)

type UciDhcpApi struct{}

func NewUciDhcpApi() *UciDhcpApi {
	return &UciDhcpApi{}
}

func (self *UciDhcpApi) GetSection(ifname string) (section string, ok bool) {
	sections, ok := UciTree.GetSections("dhcp", "dhcp")
	if !ok {
		return "", false
	}

	for _, s := range sections {
		interfaces, ok := UciTree.Get("dhcp", s, "interface")
		if ok && len(interfaces) > 0 && interfaces[0] == ifname {
			return s, true
		}
	}

	return "", false

}

func (self *UciDhcpApi) GetConfig(section string) (dhcp *sdkapi.DhcpCfg, ok bool) {
	ifaces, ok := UciTree.Get("dhcp", section, "interface")
	if !ok {
		return nil, false
	}

	ignores, ok := UciTree.Get("dhcp", section, "ignore")
	if ok && len(ignores) > 0 && ignores[0] == "1" {
		return nil, false
	}

	startIps, ok := UciTree.Get("dhcp", section, "start")
	if !ok {
		return nil, false
	}

	limits, ok := UciTree.Get("dhcp", section, "limit")
	if !ok {
		return nil, false
	}

	leases, ok := UciTree.Get("dhcp", section, "leasetime")
	if !ok {
		return nil, false
	}

	limit, err := strconv.Atoi(limits[0])
	if err != nil {
		return nil, false
	}

	leasetime, err := strconv.Atoi(strings.Replace(leases[0], "h", "", 1))
	if err != nil {
		return nil, false
	}

	return &sdkapi.DhcpCfg{
		Ifname:    ifaces[0],
		Section:   section,
		StartIp:   startIps[0],
		Limit:     uint(limit),
		LeaseHour: uint(leasetime),
	}, true

}

func (self *UciDhcpApi) SetConfig(ifname string, cfg *sdkapi.DhcpCfg) error {
	section, ok := self.GetSection(ifname)
	if !ok {
		return errors.New("Failed to get dhcp section of " + ifname)
	}

	ok = UciTree.Set("dhcp", section, "interface", cfg.Ifname)
	if !ok {
		return errors.New("Failed to set interface option of dhcp config for " + ifname)
	}

	ok = UciTree.Set("dhcp", section, "start", cfg.StartIp)
	if !ok {
		return errors.New("Failed to set start option of dhcp config for " + ifname)
	}

	ok = UciTree.Set("dhcp", section, "limit", fmt.Sprintf("%d", cfg.Limit))
	if !ok {
		return errors.New("Failed to set limit option of dhcp config for " + ifname)
	}

	ok = UciTree.Set("dhcp", section, "leasetime", fmt.Sprintf("%dh", cfg.LeaseHour))
	if !ok {
		return errors.New("Failed to set leasetime option of dhcp config for " + ifname)
	}

	return nil
}
