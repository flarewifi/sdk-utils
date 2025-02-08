package models

import (
	"context"
	"log"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/jackc/pgx/v5/pgtype"
)

type DeviceModel struct {
	db     *db.Database
	models *Models
}

func NewDeviceModel(database *db.Database, mdls *Models) *DeviceModel {
	return &DeviceModel{database, mdls}
}

func (self *DeviceModel) Create(ctx context.Context, mac string, ip string, hostname string) (*Device, error) {
	tx, err := self.db.SqlDB().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := self.db.Queries.WithTx(tx)

	dId, err := qtx.CreateDevice(ctx, queries.CreateDeviceParams{
		MacAddress: mac,
		IpAddress:  ip,
		Hostname:   hostname,
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
	}

	_, err = qtx.CreateWallet(ctx, queries.CreateWalletParams{
		DeviceID: dId,
		Balance:  sdkutils.PgFloat64ToNumeric(0),
	})
	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return dev, nil
}

func (self *DeviceModel) Find(ctx context.Context, id pgtype.UUID) (*Device, error) {
	d, err := self.db.Queries.FindDevice(ctx, id)
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
	device.macaddr = d.MacAddress
	device.ipaddr = d.IpAddress
	device.hostname = d.Hostname
	device.createdAt = d.CreatedAt.Time

	// log.Printf("Found device: %+v", device)
	return device, nil
}

func (self *DeviceModel) Update(ctx context.Context, id pgtype.UUID, mac string, ip string, hostname string) error {
	err := self.db.Queries.UpdateDevice(ctx, queries.UpdateDeviceParams{
		ID:         id,
		MacAddress: mac,
		IpAddress:  ip,
		Hostname:   hostname,
	})
	if err != nil {
		log.Printf("error updating device %v: %v", id, err)
		return err
	}

	log.Printf("Successfully updated device with id %v", id)
	return nil
}
