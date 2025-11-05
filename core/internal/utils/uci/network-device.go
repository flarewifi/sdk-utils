package uci

import "errors"

// device
func (self *UciNetworkApi) GetDevice(section string) (dev string, err error) {
	devices, ok := UciTree.Get("network", section, "device")
	if !ok || len(devices) < 1 {
		return "", errors.New("Cant get device " + section)
	}

	return devices[0], nil
}

func (self *UciNetworkApi) GetDeviceSec(name string) (section string, err error) {
	defErr := errors.New("Can't get device section for device " + name)
	sections, ok := UciTree.GetSections("network", "device")
	if !ok {
		return "", defErr
	}

	for _, sec := range sections {
		names, ok := UciTree.Get("network", sec, "name")
		if ok && len(names) > 0 && names[0] == name {
			return sec, nil
		}
	}

	return "", defErr
}

func (self *UciNetworkApi) GetDeviceType(section string) (t string, err error) {
	types, ok := UciTree.Get("network", section, "type")
	if ok && len(types) > 0 {
		return types[0], nil
	}

	return "", errors.New("Can't get type of section " + section)
}
