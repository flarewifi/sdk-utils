package uci

import (
	"errors"
	"fmt"
	"strconv"

	sdkapi "sdk/api"
)

type UciWirelessApi struct{}

func NewUciWirelessApi() *UciWirelessApi {
	return &UciWirelessApi{}
}

func (self *UciWirelessApi) GetDeviceSecs() (sections []string) {
	secs, _ := UciTree.GetSections("wireless", "wifi-device")
	return secs
}

func (self *UciWirelessApi) GetIfaceSecs() (sections []string) {
	secs, _ := UciTree.GetSections("wireless", "wifi-iface")
	return secs
}

func (self *UciWirelessApi) GetDevice(section string) (dev *sdkapi.WifiDev, err error) {
	var wdev sdkapi.WifiDev

	wdev.Section = section

	types, ok := UciTree.Get("wireless", section, "type")
	if !ok {
		return nil, errors.New("Cannot get type of wireless section " + section)
	}

	if len(types) > 0 {
		wdev.Type = types[0]
	}

	paths, ok := UciTree.Get("wireless", section, "path")
	if ok && len(paths) > 0 {
		wdev.Path = paths[0]
	}

	chans, ok := UciTree.Get("wireless", section, "channel")
	if ok && len(chans) > 0 {
		ch, err := strconv.Atoi(chans[0])
		if err != nil {
			return nil, err
		}
		wdev.Channel = uint(ch)
	}

	bands, ok := UciTree.Get("wireless", section, "band")
	if ok && len(bands) > 0 {
		wdev.Band = bands[0]
	}

	htmodes, ok := UciTree.Get("wireless", section, "htmode")
	if ok && len(htmodes) > 0 {
		wdev.Htmode = htmodes[0]
	}

	dis, ok := UciTree.Get("wireless", section, "disabled")
	wdev.Disabled = ok && len(dis) > 0 && dis[0] == "1"

	return &wdev, nil
}

func (self *UciWirelessApi) GetIface(section string) (iface *sdkapi.WifiIface, err error) {
	var wlan sdkapi.WifiIface

	wlan.Section = section

	devices, ok := UciTree.Get("wireless", section, "device")
	if !ok {
		return nil, errors.New("Unable to get device of wireless section " + section)
	}

	if len(devices) > 0 {
		wlan.Device = devices[0]
	}

	nets, ok := UciTree.Get("wireless", section, "network")
	if ok && len(nets) > 0 {
		wlan.Network = nets[0]
	}

	modes, ok := UciTree.Get("wireless", section, "mode")
	if ok && len(modes) > 0 {
		wlan.Mode = modes[0]
	}

	ssids, ok := UciTree.Get("wireless", section, "ssid")
	if ok && len(ssids) > 0 {
		wlan.Ssid = ssids[0]
	}

	encs, ok := UciTree.Get("wireless", section, "encryption")
	if ok && len(encs) > 0 {
		wlan.Encryption = encs[0]
	}

	keys, ok := UciTree.Get("wireless", section, "key")
	if ok && len(keys) > 0 {
		wlan.Key = keys[0]
	}

	return &wlan, nil
}

func (self *UciWirelessApi) GetDevices() (devs []*sdkapi.WifiDev) {
	secs := self.GetDeviceSecs()

	for _, s := range secs {
		dev, err := self.GetDevice(s)
		if err == nil {
			devs = append(devs, dev)
		}
	}

	return devs
}

func (self *UciWirelessApi) GetIfaces() (ifaces []*sdkapi.WifiIface) {
	secs := self.GetIfaceSecs()

	for _, s := range secs {
		iface, err := self.GetIface(s)
		if err == nil {
			ifaces = append(ifaces, iface)
		}
	}

	return ifaces
}

func (self *UciWirelessApi) SetDevice(section string, cfg *sdkapi.WifiDev) error {
	var ok bool

	disabled := "0"
	if cfg.Disabled {
		disabled = "1"
	}

	if ok = cfg.Type != "" && UciTree.Set("wireless", section, "type", cfg.Type); !ok {
		return errors.New("Failed to set device option of wireless section " + section)
	}

	if ok = cfg.Path != "" && UciTree.Set("wireless", section, "path", cfg.Path); !ok {
		return errors.New("Failed to set path option of wireless section " + section)
	}

	if ok = cfg.Channel > 0 && UciTree.Set("wireless", section, "channel", fmt.Sprintf("%d", cfg.Channel)); !ok {
		return errors.New("Failed to set channel option of wireless section " + section)
	}

	if ok = cfg.Band != "" && UciTree.Set("wireless", section, "band", cfg.Band); !ok {
		return errors.New("Failed to set band option of wireless section " + section)
	}

	if ok = cfg.Htmode != "" && UciTree.Set("wireless", section, "htmode", cfg.Htmode); !ok {
		return errors.New("Failed to set htmode option of wireless section " + section)
	}

	if ok = UciTree.Set("wireless", section, "disabled", disabled); !ok {
		return errors.New("Failed to set disabled option of wireless section " + section)
	}

	return nil
}

func (self *UciWirelessApi) SetIface(section string, cfg *sdkapi.WifiIface) error {
	var ok bool

	if cfg.Device != "" {
		if ok = UciTree.Set("wireless", section, "device", cfg.Device); !ok {
			return errors.New("Failed to set device option of wireless section " + section)
		}
	}

	if cfg.Network != "" {
		if ok = UciTree.Set("wireless", section, "network", cfg.Network); !ok {
			return errors.New("Failed to set network option of wireless section " + section)
		}
	}

	if cfg.Mode != "" {
		if ok = UciTree.Set("wireless", section, "mode", cfg.Mode); !ok {
			return errors.New("Failed to set network option of wireless section " + section)
		}
	}

	if cfg.Ssid != "" {
		if ok = UciTree.Set("wireless", section, "ssid", cfg.Ssid); !ok {
			return errors.New("Failed to set ssid option of wireless section " + section)
		}
	}

	if cfg.Encryption != "" {
		if ok = UciTree.Set("wireless", section, "encryption", cfg.Encryption); !ok {
			return errors.New("Failed to set encryption option of wireless section " + section)
		}
	}

	// TODO: needs to evaluate key and other options base on encryption mode
	if cfg.Encryption != "none" && cfg.Key == "" {
		return errors.New("WiFi key cannot be empty.")
	}

	if cfg.Encryption != "none" && cfg.Key != "" {
		if ok = UciTree.Set("wireless", section, "key", cfg.Key); !ok {
			return errors.New("Failed to set key option of wireless section " + section)
		}
	} else {
		UciTree.Del("wireless", section, "key")
	}

	return nil
}
