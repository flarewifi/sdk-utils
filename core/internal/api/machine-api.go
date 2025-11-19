package api

import (
	machineuid "core/internal/utils/machine-uid"
)

type MachineApi struct {
	api *PluginApi
}

func NewMachineApi(api *PluginApi) {
	machineApi := &MachineApi{api: api}
	api.MachineAPI = machineApi
}

func (m *MachineApi) MacineID() string {
	return machineuid.GetMachineUID()
}
