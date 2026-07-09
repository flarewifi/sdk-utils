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

// FindClientById finds a client device by its ID.
func (self *ClientsMgrApi) FindClientById(ctx context.Context, devId int64) (sdkapi.IClientDevice, error) {
	return self.pluginApi.SessionMgr.FindDeviceByID(ctx, devId)
}

// FindClientByMac finds a client device by its MAC address.
func (self *ClientsMgrApi) FindClientByMac(ctx context.Context, mac string) (sdkapi.IClientDevice, error) {
	return self.pluginApi.SessionMgr.FindClientByMac(ctx, mac)
}

// FindClientByIp finds a client device by its IP address.
func (self *ClientsMgrApi) FindClientByIp(ctx context.Context, ip string) (sdkapi.IClientDevice, error) {
	return self.pluginApi.SessionMgr.FindClientByIp(ctx, ip)
}

// FindClientByUUID finds a client device by its globally unique identifier.
func (self *ClientsMgrApi) FindClientByUUID(ctx context.Context, uuid string) (sdkapi.IClientDevice, error) {
	return self.pluginApi.SessionMgr.FindDeviceByUUID(ctx, uuid)
}

// NewClientDevice wraps device data into an IClientDevice object without performing
// additional database queries. This is useful when you already have device data from queries
// and want to use SDK methods like Update(), Emit(), and Subscribe(). The params parameter
// contains all device fields from the database row. Also use this to build an in-memory
// preview (e.g. ID left at 0) to pass to RegisterClient.
func (self *ClientsMgrApi) NewClientDevice(params sdkapi.NewDeviceParams) sdkapi.IClientDevice {
	return self.pluginApi.SessionMgr.NewClientDevice(params)
}

// RegisterClient persists dev (built via NewClientDevice) as a real device
// record. Unlike the live captive-portal registration flow, the caller
// already knows the exact MAC/IP/hostname to register, so no
// cookie/fingerprint/ARP-NDP disambiguation is performed here.
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

// MergeClientDevices merges the source device into the target device.
func (self *ClientsMgrApi) MergeClientDevices(ctx context.Context, targetID, sourceID int64) error {
	return self.pluginApi.SessionMgr.MergeClientDevices(ctx, targetID, sourceID)
}
