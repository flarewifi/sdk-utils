package uci

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// bridge-vlan
func (self *UciNetworkApi) CreateBrVlan(brvlan *sdkapi.BrVlan, ports []*sdkapi.BrVlanPort) error {
	sec := sdkutils.RandomStr(16)

	if _, ok := self.GetBrVlanSec(brvlan); !ok {
		if err := UciTree.AddSection("network", sec, "bridge-vlan"); err != nil {
			return err
		}
	}

	ok := UciTree.Set("network", sec, "device", brvlan.Device)
	if !ok {
		return errors.New("Failed to set device option of " + sec)
	}

	ok = UciTree.Set("network", sec, "vlan", fmt.Sprintf("%d", brvlan.VlanID))
	if !ok {
		return errors.New("Failed to set vlan option of " + sec)
	}

	var portTags []string
	for _, p := range ports {
		portTags = append(portTags, p.String())
	}

	ok = UciTree.Set("network", sec, "ports", portTags...)
	if !ok {
		return errors.New("Failed to set ports list of " + sec)
	}

	return nil

}

func (self *UciNetworkApi) GetBrVlanSecs() (sections []string) {
	sections, ok := UciTree.GetSections("network", "bridge-vlan")
	if !ok {
		return []string{}
	}
	return sections
}

func (self *UciNetworkApi) GetBrVlanSec(brvlan *sdkapi.BrVlan) (section string, ok bool) {
	sections, ok := UciTree.GetSections("network", "bridge-vlan")
	if !ok {
		return "", false
	}

	for _, sec := range sections {
		devices, ok := UciTree.Get("network", sec, "device")
		devOK := ok && len(devices) > 0 && devices[0] == brvlan.Device

		vlans, ok := UciTree.Get("network", sec, "vlan")
		vlanOK := ok && len(vlans) > 0 && vlans[0] == fmt.Sprintf("%d", brvlan.VlanID)

		if devOK && vlanOK {
			return sec, true
		}
	}

	return "", false
}

func (self *UciNetworkApi) GetBrVlanID(brvlan *sdkapi.BrVlan) (vlanid int, ok bool) {
	sec, ok := self.GetBrVlanSec(brvlan)
	if !ok {
		return 0, false
	}

	vlans, ok := UciTree.Get("network", sec, "vlan")
	if ok && len(vlans) > 0 {
		vlan, err := strconv.Atoi(vlans[0])
		if err != nil {
			return 0, false
		}
		return vlan, true
	}

	return 0, false
}

func (self *UciNetworkApi) GetBrVlanPorts(brvlan *sdkapi.BrVlan) (ports []*sdkapi.BrVlanPort, err error) {
	sec, ok := self.GetBrVlanSec(brvlan)
	if !ok {
		return nil, sdkapi.ErrNotBrVlan
	}

	ethTags, ok := UciTree.Get("network", sec, "port")
	if ok {
		for _, d := range ethTags {
			var port sdkapi.BrVlanPort

			ethTag := strings.Split(d, ":")

			if len(ethTag) == 2 {
				eth, tag := ethTag[0], ethTag[1]
				port.Device = eth

				if tag == "u*" {
					port.Untagged = true
					port.Primary = true
				} else if tag == "u" {
					port.Untagged = true
				} else if tag == "t" {
					port.Tagged = true
				} else {
					port.Tagged = true
				}
			} else {
				port.Device = ethTag[0]
				port.Tagged = true
			}

			ports = append(ports, &port)
		}

		return ports, nil
	}

	return nil, sdkapi.ErrNotBrVlan

}

func (self *UciNetworkApi) SetBrVlanPorts(brvlan *sdkapi.BrVlan, ports []*sdkapi.BrVlanPort) error {
	sec, ok := self.GetBrVlanSec(brvlan)
	if !ok {
		return sdkapi.ErrNotBrVlan
	}

	var portstr []string
	for _, p := range ports {
		portstr = append(portstr, p.String())
	}

	ok = UciTree.Set("network", sec, "ports", portstr...)
	if !ok {
		return fmt.Errorf("Failed to set bridge-vlan ports of %s. Check if this section exists.", brvlan)
	}

	return nil

}

func (self *UciNetworkApi) DeleteBrVlan(brvlan *sdkapi.BrVlan) {
	if sec, ok := self.GetBrVlanSec(brvlan); ok {
		UciTree.DelSection("network", sec)
	}
}
