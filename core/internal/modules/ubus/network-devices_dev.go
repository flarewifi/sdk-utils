//go:build dev

package ubus

import "errors"

var (
	brlan = NetworkDevice{
		Name:          "br-lan",
		Type:          "bridge",
		Up:            true,
		Speed:         "1000F",
		BridgeMembers: []string{"eth0"},
	}

	eth0 = NetworkDevice{
		Name:  "eth0",
		Type:  "ethernet",
		Up:    true,
		Speed: "1000F",
	}

	eth1 = NetworkDevice{
		Name:  "eth1",
		Type:  "ethernet",
		Up:    true,
		Speed: "1000F",
	}

	wlan = NetworkDevice{
		Name:  "wlan0",
		Type:  "wlan",
		Up:    true,
		Speed: "1000F",
	}

	vlan = NetworkDevice{
		Name:  "br-lan.22",
		Type:  "vlan",
		Up:    true,
		Speed: "1000F",
	}
)

func GetNetworkDevices() ([]*NetworkDevice, error) {
	return []*NetworkDevice{&brlan, &eth0, &eth1, &wlan, &vlan}, nil
}

func GetNetworkDevice(device string) (*NetworkDevice, error) {
	if device == "br-lan" {
		return &brlan, nil
	}
	if device == "eth0" {
		return &eth0, nil
	}
	if device == "eth1" {
		return &eth1, nil
	}
	if device == "wlan0" {
		return &wlan, nil
	}
	if device == "br-lan.22" {
		return &vlan, nil
	}
	return nil, errors.New("Mock device for " + device + " is not implemeted.")
}
