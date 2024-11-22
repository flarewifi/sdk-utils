//go:build !dev

package ubus

import (
	"bytes"
	"fmt"
	"log"

	"github.com/goccy/go-json"

	"core/internal/utils/cmd"
)

func GetNetworkDevices() ([]*NetworkDevice, error) {
	var out bytes.Buffer
	err := cmd.ExecOutput("ubus call network.device status", &out)
	if err != nil {
		return nil, err
	}

	var devStat map[string]*NetworkDevice
	err = json.Unmarshal(out.Bytes(), &devStat)
	if err != nil {
		return nil, err
	}

	var devices []*NetworkDevice
	for k := range devStat {
		devStat[k].Name = k
		devices = append(devices, devStat[k])
	}

	return devices, nil
}

func GetNetworkDevice(device string) (*NetworkDevice, error) {
	var out bytes.Buffer
	err := cmd.ExecOutput(fmt.Sprintf("ubus call network.device status {\"name\":\"%s\"}", device), &out)
	if err != nil {
		return nil, err
	}
	var dev NetworkDevice
	err = json.Unmarshal(out.Bytes(), &dev)
	dev.Name = device
	log.Println(dev)
	return &dev, err
}
