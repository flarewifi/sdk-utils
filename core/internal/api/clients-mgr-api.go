package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"core/db/models"
	"core/internal/sessmgr"
	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
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

	created, err := self.pluginApi.models.Device().Create(ctx, nil, models.CreateDeviceParams{
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

// BatchRegisterClient persists a batch of devices (built via NewClientDevice) as
// real device records in a single transaction. See IClientsMgrApi for the full
// event/rollback contract.
func (self *ClientsMgrApi) BatchRegisterClient(clnts []sdkapi.IClientDevice) error {
	if len(clnts) == 0 {
		return nil
	}

	ctx := context.Background()

	for _, clnt := range clnts {
		if clnt.Data().MacAddr == "" {
			return fmt.Errorf("mac address cannot be blank")
		}
	}

	// Reject any MAC that has ever been registered before firing a single event
	// or opening the transaction — same check, and same ordering relative to
	// events, as the single-item RegisterClient path. Doing this first means a
	// batch that's going to fail on a duplicate MAC fails before any before-create
	// side effects run for ANY device in the batch, not just the offending one.
	for _, clnt := range clnts {
		data := clnt.Data()
		if existingID, err := self.pluginApi.models.DeviceMac().FindDeviceByAnyMac(ctx, data.MacAddr); err == nil && existingID > 0 {
			return fmt.Errorf("mac %s: %w", data.MacAddr, sdkapi.ErrClientAlreadyRegistered)
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check existing mac %s: %w", data.MacAddr, err)
		}
	}

	// Batch-level before-create event: fires once before any DB writes. An error
	// here cancels the whole batch with no rollback needed.
	if err := self.pluginApi.EventsMgr.EmitClientBatchEvent(ctx, sdkapi.EventClientBatchBeforeCreate, clnts); err != nil {
		return err
	}

	// Per-device before-create events fire here, BEFORE the transaction opens —
	// not inside the RunInTx below. This app runs SQLite through a single shared
	// connection (db.SetMaxOpenConns(1)): if a subscriber's callback made its own
	// DB call while our transaction held that one connection, it would block
	// forever waiting for a connection only our own (blocked) goroutine could
	// free. Firing here, with no transaction open, means a subscriber's query is
	// safe, and a veto needs no rollback since nothing has been written yet.
	for _, clnt := range clnts {
		if err := self.pluginApi.SessionMgr.EmitClientEvent(ctx, sdkapi.EventClientBeforeCreate, clnt); err != nil {
			return fmt.Errorf("client registration vetoed for mac %s: %w", clnt.Data().MacAddr, err)
		}
	}

	var created []sdkapi.IClientDevice
	err := sdkutils.RunInTx(self.pluginApi.db.DB, ctx, func(tx *sql.Tx) error {
		for _, clnt := range clnts {
			data := clnt.Data()

			// Route through the same validated Device().Create path as the
			// single-item RegisterClient (tx-scoped here so the whole batch
			// commits or rolls back together) — this enforces the same
			// required-field checks (e.g. at least one IP address) instead of
			// inserting the row directly and skipping them.
			dev, err := self.pluginApi.models.Device().Create(ctx, tx, models.CreateDeviceParams{
				MacAddress:  data.MacAddr,
				Ipv4Address: data.Ipv4Addr,
				Ipv6Address: data.Ipv6Addr,
				Hostname:    data.Hostname,
			})
			if err != nil {
				return fmt.Errorf("create device for mac %s: %w", data.MacAddr, err)
			}
			created = append(created, sessmgr.NewClientDevice(self.pluginApi.db, self.pluginApi.models, self.pluginApi.SessionMgr, dev))
		}

		return nil
	})
	if err != nil {
		return err
	}

	for _, clnt := range created {
		self.pluginApi.SessionMgr.EmitClientEvent(ctx, sdkapi.EventClientCreated, clnt)
		self.pluginApi.SessionMgr.EmitClientEvent(ctx, sdkapi.EventClientRegistered, clnt)
	}
	self.pluginApi.EventsMgr.EmitClientBatchEvent(ctx, sdkapi.EventClientBatchCreated, created)

	return nil
}

// MergeClientDevices merges the source device into the target device.
func (self *ClientsMgrApi) MergeClientDevices(ctx context.Context, targetID, sourceID int64) error {
	return self.pluginApi.SessionMgr.MergeClientDevices(ctx, targetID, sourceID)
}
