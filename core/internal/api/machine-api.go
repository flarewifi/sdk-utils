package api

import (
	machineuid "core/internal/modules/machine-uid"
	"core/internal/modules/netmon"
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
