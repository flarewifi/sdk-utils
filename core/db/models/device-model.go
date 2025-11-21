package models

import (
	"context"
	"database/sql"
	"log"
	sdkapi "sdk/api"

	"core/db"
	"core/db/queries"
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
	Status     int
}

func NewDeviceModel(database *db.Database, mdls *Models) *DeviceModel {
	return &DeviceModel{db: database, models: mdls}
}

func (self *DeviceModel) Create(tx *sql.Tx, ctx context.Context, params CreateDeviceParams) (*Device, error) {
	qtx := self.db.Queries.WithTx(tx)
	dId, err := qtx.CreateDevice(ctx, queries.CreateDeviceParams{
		MacAddress: params.MacAddress,
		IpAddress:  params.IpAddress,
		Hostname:   params.Hostname,
	})
	if err != nil {
		log.Println("error creating new device:", err)
		return nil, err
	}

	d, err := qtx.FindDevice(ctx, dId)
	if err != nil {
		log.Printf("error finding device %v: %v\n", dId, err)
		return nil, err
	}

	dev := &Device{
		db:        self.db,
		models:    self.models,
		id:        d.ID,
		macaddr:   d.MacAddress,
		ipaddr:    d.IpAddress,
		hostname:  d.Hostname,
		createdAt: d.CreatedAt.Time,
		status:    sdkapi.DeviceStatus(d.Status),
	}

	_, err = qtx.CreateWallet(ctx, queries.CreateWalletParams{
		DeviceID: dId,
		Balance:  0.0,
	})
	if err != nil {
		return nil, err
	}

	return dev, nil
}

func (self *DeviceModel) Find(tx *sql.Tx, ctx context.Context, id int64) (*Device, error) {
	var d queries.FindDeviceRow
	var err error
	if tx != nil {
		qtx := self.db.Queries.WithTx(tx)
		d, err = qtx.FindDevice(ctx, id)
	} else {
		d, err = self.db.Queries.FindDevice(ctx, id)
	}
	if err != nil {
		log.Printf("error finding device %v: %v", id, err)
		return nil, err
	}

	device := NewDevice(self.db, self.models)
	device.id = d.ID
	device.macaddr = d.MacAddress
	device.ipaddr = d.IpAddress
	device.hostname = d.Hostname
	device.createdAt = d.CreatedAt.Time
	device.status = sdkapi.DeviceStatus(d.Status)

	// log.Printf("Found device: %+v", device)
	return device, nil
}

func (self *DeviceModel) FindByMac(tx *sql.Tx, ctx context.Context, mac string) (*Device, error) {
	qtx := self.db.Queries.WithTx(tx)
	device := NewDevice(self.db, self.models)
	d, err := qtx.FindDeviceByMac(ctx, mac)
	if err != nil {
		log.Printf("error finding device %s: %v", mac, err)
		return nil, err
	}

	device.id = d.ID
	device.macaddr = d.MacAddress
	device.ipaddr = d.IpAddress
	device.hostname = d.Hostname
	device.createdAt = d.CreatedAt.Time
	device.status = sdkapi.DeviceStatus(d.Status)

	return device, nil
}

func (self *DeviceModel) Update(tx *sql.Tx, ctx context.Context, params UpdateDeviceParams) error {
	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdateDevice(ctx, queries.UpdateDeviceParams{
		ID:         params.ID,
		MacAddress: params.MacAddress,
		IpAddress:  params.IpAddress,
		Hostname:   params.Hostname,
		Status:     int64(params.Status),
	})
	if err != nil {
		log.Printf("error updating device %v: %v", params.ID, err)
		return err
	}

	log.Printf("Successfully updated device with id %v", params.ID)
	return nil
}
