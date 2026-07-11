package api

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/modules/netmon"
	"core/utils/product"
	"core/utils/sysinfo"
	"sync"

	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// osReleaseFile mirrors the same literal path independently defined in the
// activation/updates modules (no shared paths package exists for this).
const osReleaseFile = "/etc/os_release.json"

// deviceModelOnce/cachedDeviceModel cache os_release.json's device_model for the
// process lifetime — it's written once at OS-image build/flash time and never
// rewritten by an OTA (see sdkutils.OsRelease's doc comment), so re-reading the
// file on every call would be wasted I/O for a value that can't change.
var (
	deviceModelOnce   sync.Once
	cachedDeviceModel string
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

// DeviceModel returns the machine's board/device model, read from the frozen
// /etc/os_release.json. See IMachineApi.DeviceModel.
func (m *MachineApi) DeviceModel() string {
	deviceModelOnce.Do(func() {
		release, err := sdkutils.ReadOsRelease(osReleaseFile)
		if err != nil {
			return
		}
		cachedDeviceModel = release.DeviceModel
	})
	return cachedDeviceModel
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
