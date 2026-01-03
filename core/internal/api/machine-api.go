package api

import (
	machineuid "core/internal/modules/machine-uid"
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
