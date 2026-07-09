package api

import (
	"context"
	"fmt"

	"core/db/models"
	"core/internal/sessmgr"
	sdkapi "sdk/api"
)

func NewClientsMgrApi(pluginApi *PluginApi) *ClientsMgrApi {
	clientsMgrApi := &ClientsMgrApi{
		pluginApi: pluginApi,
	}
	pluginApi.ClientsMgrAPI = clientsMgrApi
	return clientsMgrApi
}

type ClientsMgrApi struct {
	pluginApi *PluginApi
}

// RegisterClient persists a client device preview (built via
// SessionsMgr().NewClientDevice) as a real device record. Unlike the live
// captive-portal registration flow, the caller already knows the exact
// MAC/IP/hostname to register, so no cookie/fingerprint/ARP-NDP
// disambiguation is performed here.
func (self *ClientsMgrApi) RegisterClient(dev sdkapi.IClientDevice) error {
	ctx := context.Background()
	data := dev.Data()

	if data.MacAddr == "" {
		return fmt.Errorf("mac address cannot be blank")
	}

	if existingID, err := self.pluginApi.models.DeviceMac().FindDeviceByAnyMac(ctx, data.MacAddr); err == nil && existingID > 0 {
		return sdkapi.ErrClientAlreadyRegistered
	}

	if err := self.pluginApi.SessionMgr.EmitClientEvent(ctx, sdkapi.EventClientBeforeCreate, dev); err != nil {
		return fmt.Errorf("client registration vetoed: %w", err)
	}

	created, err := self.pluginApi.models.Device().Create(ctx, models.CreateDeviceParams{
		MacAddress:  data.MacAddr,
		Ipv4Address: data.Ipv4Addr,
		Ipv6Address: data.Ipv6Addr,
		Hostname:    data.Hostname,
	})
	if err != nil {
		return fmt.Errorf("create device: %w", err)
	}

	clnt := sessmgr.NewClientDevice(self.pluginApi.db, self.pluginApi.models, self.pluginApi.SessionMgr, created)
	self.pluginApi.SessionMgr.EmitClientEvent(ctx, sdkapi.EventClientCreated, clnt)
	self.pluginApi.SessionMgr.EmitClientEvent(ctx, sdkapi.EventClientRegistered, clnt)

	return nil
}
