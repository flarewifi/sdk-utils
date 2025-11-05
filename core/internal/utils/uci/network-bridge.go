package uci

import (
	"errors"
	"fmt"
	"log"

	sdkapi "sdk/api"
)

// bridge
func (self *UciNetworkApi) GetBridgeSecs() (sections []string) {
	secs, ok := UciTree.GetSections("network", "device")
	if !ok {
		return sections
	}

	for _, sec := range secs {
		t, err := self.GetDeviceType(sec)
		if err == nil && t == "bridge" {
			sections = append(sections, sec)
		}
	}

	return sections
}

func (self *UciNetworkApi) GetBridgeVlanFilter(section string) (enabled bool) {
	vals, ok := UciTree.Get("network", section, "vlan_filtering")
	if ok && len(vals) > 0 {
		return vals[0] == "1"
	}

	return false
}

func (self *UciNetworkApi) GetBridgePorts(section string) (ports []string, err error) {
	types, ok := UciTree.Get("network", section, "type")
	if !ok {
		log.Println("Device " + section + " type is not a bridge")
		return nil, sdkapi.ErrNotBridge
	}

	if len(types) == 0 || (len(types) > 0 && types[0] != "bridge") {
		return nil, sdkapi.ErrNotBridge
	}

	ports, _ = UciTree.Get("network", section, "ports")

	return ports, nil
}

func (self *UciNetworkApi) SetBridgePorts(section string, ports []string) error {
	types, ok := UciTree.Get("network", section, "type")
	if !ok {
		return sdkapi.ErrNotBridge
	}

	if len(types) > 0 && types[0] != "bridge" {
		return sdkapi.ErrNotBridge
	}

	ok = UciTree.Set("network", section, "ports", ports...)
	if !ok {
		return errors.New("Can't set bridge ports for section " + section)
	}

	return nil
}

func (self *UciNetworkApi) SetBridgeVlanFilter(section string, enabled bool) error {
	val := "0"
	if enabled {
		val = "1"
	}

	ok := UciTree.Set("network", section, "vlan_filtering", val)
	if !ok {
		return fmt.Errorf("Failed to set vlan_filtering option of section %s", section)
	}

	return nil
}
