package api

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/modules/netmon"
	"core/utils/product"
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
