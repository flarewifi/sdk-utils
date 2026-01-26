package models

import (
	"context"
	"fmt"
	"log"
	"strings"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
	sdkapi "sdk/api"
)

type DeviceModel struct {
	db     *db.Database
	models *Models
}

// CreateDeviceParams holds parameters for creating a new device
type CreateDeviceParams struct {
	MacAddress string
	IpAddress  string
	Hostname   string
}

// UpdateDeviceParams holds parameters for updating a device
type UpdateDeviceParams struct {
	ID         int64
	MacAddress string
	IpAddress  string
	Hostname   string
	UUID       string
	Status     int
}

func NewDeviceModel(database *db.Database, mdls *Models) *DeviceModel {
	return &DeviceModel{db: database, models: mdls}
}

// validateDeviceFields checks that required device fields are not blank
func validateDeviceFields(uuid, ip, mac string) error {
	if strings.TrimSpace(uuid) == "" {
		return fmt.Errorf("uuid cannot be blank")
	}
	if strings.TrimSpace(ip) == "" {
		return fmt.Errorf("ip address cannot be blank")
	}
	if strings.TrimSpace(mac) == "" {
		return fmt.Errorf("mac address cannot be blank")
	}
	return nil
}

func (self *DeviceModel) Create(ctx context.Context, params CreateDeviceParams) (*Device, error) {
	uid := sdkutils.NewUUID()

	// Validate required fields
	if err := validateDeviceFields(uid, params.IpAddress, params.MacAddress); err != nil {
		log.Printf("device validation failed: %v", err)
		return nil, err
	}

	dId, err := self.db.Queries.CreateDevice(ctx, queries.CreateDeviceParams{
		MacAddress: params.MacAddress,
		IpAddress:  params.IpAddress,
		Hostname:   params.Hostname,
		Uuid:       uid,
	})
	if err != nil {
		log.Println("error creating new device:", err)
		return nil, err
	}

	d, err := self.db.Queries.FindDevice(ctx, dId)
	if err != nil {
		log.Printf("error finding device %v: %v\n", dId, err)
		return nil, err
	}

	dev := &Device{
		db:        self.db,
		models:    self.models,
		id:        d.ID,
		uuid:      d.Uuid,
		macaddr:   d.MacAddress,
		ipaddr:    d.IpAddress,
		hostname:  d.Hostname,
		createdAt: d.CreatedAt.Time,
		status:    sdkapi.DeviceStatus(d.Status),
	}

	_, err = self.db.Queries.CreateWallet(ctx, queries.CreateWalletParams{
		DeviceID: dId,
		Balance:  0.0,
	})
	if err != nil {
		return nil, err
	}

	return dev, nil
}

func (self *DeviceModel) Find(ctx context.Context, id int64) (*Device, error) {
	d, err := self.db.Queries.FindDevice(ctx, id)
	if err != nil {
		log.Printf("error finding device %v: %v", id, err)
		return nil, err
	}

	device := NewDevice(self.db, self.models)
	device.id = d.ID
	device.uuid = d.Uuid
	device.macaddr = d.MacAddress
	device.ipaddr = d.IpAddress
	device.hostname = d.Hostname
	device.createdAt = d.CreatedAt.Time
	device.status = sdkapi.DeviceStatus(d.Status)

	// log.Printf("Found device: %+v", device)
	return device, nil
}

func (self *DeviceModel) FindByMac(ctx context.Context, mac string) (*Device, error) {
	device := NewDevice(self.db, self.models)
	d, err := self.db.Queries.FindDeviceByMac(ctx, mac)
	if err != nil {
		log.Printf("error finding device %s: %v", mac, err)
		return nil, err
	}

	device.id = d.ID
	device.uuid = d.Uuid
	device.macaddr = d.MacAddress
	device.ipaddr = d.IpAddress
	device.hostname = d.Hostname
	device.createdAt = d.CreatedAt.Time
	device.status = sdkapi.DeviceStatus(d.Status)

	return device, nil
}

func (self *DeviceModel) FindByUUID(ctx context.Context, uid string) (*Device, error) {
	device := NewDevice(self.db, self.models)
	d, err := self.db.Queries.FindDeviceByUUID(ctx, uid)
	if err != nil {
		log.Printf("error finding device by UUID %s: %v", uid, err)
		return nil, err
	}

	device.id = d.ID
	device.uuid = d.Uuid
	device.macaddr = d.MacAddress
	device.ipaddr = d.IpAddress
	device.hostname = d.Hostname
	device.createdAt = d.CreatedAt.Time
	device.status = sdkapi.DeviceStatus(d.Status)

	return device, nil
}

func (self *DeviceModel) Update(ctx context.Context, params UpdateDeviceParams) error {
	// Validate required fields
	if err := validateDeviceFields(params.UUID, params.IpAddress, params.MacAddress); err != nil {
		log.Printf("device validation failed: %v", err)
		return err
	}

	err := self.db.Queries.UpdateDevice(ctx, queries.UpdateDeviceParams{
		ID:         params.ID,
		MacAddress: params.MacAddress,
		IpAddress:  params.IpAddress,
		Hostname:   params.Hostname,
		Uuid:       params.UUID,
		Status:     int64(params.Status),
	})
	if err != nil {
		log.Printf("error updating device %v: %v", params.ID, err)
		return err
	}

	log.Printf("Successfully updated device with id %v", params.ID)
	return nil
}

// BackfillEmptyUUIDs generates UUIDs for all devices that have empty UUID fields
func (self *DeviceModel) BackfillEmptyUUIDs(ctx context.Context) error {
	devices, err := self.db.Queries.FindDevicesWithEmptyUUID(ctx)
	if err != nil {
		log.Printf("error finding devices with empty UUID: %v", err)
		return err
	}

	for _, d := range devices {
		uid := sdkutils.NewUUID()
		err := self.db.Queries.UpdateDeviceUUID(ctx, queries.UpdateDeviceUUIDParams{
			ID:   d.ID,
			Uuid: uid,
		})
		if err != nil {
			log.Printf("error updating UUID for device %v: %v", d.ID, err)
			return err
		}
		log.Printf("Generated UUID %s for device %v", uid, d.ID)
	}

	if len(devices) > 0 {
		log.Printf("Backfilled UUIDs for %d devices", len(devices))
	}

	return nil
}
