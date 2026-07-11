package api

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/modules/netmon"
	"core/utils/product"
	"core/utils/sysinfo"

	sdkapi "sdk/api"
)

type MachineApi struct {
	api *PluginApi
}

func NewMachineApi(api *PluginApi) {
	machineApi := &MachineApi{api: api}
	api.MachineAPI = machineApi
}

func (m *MachineApi) GetID() string {
	_, machineID := machineuid.GetMachineUID()
	return machineID
}

// IsOnline reports whether the machine currently has internet access, as observed
// by the core's online monitor (the same signal behind EventInternetUp/Down).
func (m *MachineApi) IsOnline() bool {
	return netmon.IsOnline()
}

// ProductVersion returns the per-B2B-partner product version stamped into
// core/product.json by the software-release build (falling back to the core
// version when unstamped). See IMachineApi.ProductVersion.
func (m *MachineApi) ProductVersion() string {
	return product.Version()
}

// DeviceModel returns the machine's board/device model, decrypted from
// core/product.json. See IMachineApi.DeviceModel.
func (m *MachineApi) DeviceModel() string {
	return product.DeviceModel()
}

// SystemStats returns a snapshot of the machine's current CPU, memory, disk,
// and temperature usage. See IMachineApi.SystemStats.
func (m *MachineApi) SystemStats() sdkapi.SystemStats {
	stats := sdkapi.SystemStats{}

	info, err := sysinfo.GetSystemInfo()
	if err != nil || info == nil {
		return stats
	}

	if len(info.CPUPercent) > 0 {
		var sum float64
		for _, p := range info.CPUPercent {
			sum += p
		}
		stats.CpuPercent = sum / float64(len(info.CPUPercent))
	}

	stats.MemTotal = info.MemTotal
	stats.MemUsed = info.MemUsed
	stats.DiskTotal = info.DiskTotal
	stats.DiskUsed = info.DiskUsed

	if len(info.CPUTemperature) > 0 {
		var sum float64
		for _, t := range info.CPUTemperature {
			sum += t
		}
		avg := sum / float64(len(info.CPUTemperature))
		stats.TemperatureCelsius = &avg
	}

	return stats
}
