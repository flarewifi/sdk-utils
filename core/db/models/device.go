package models

import (
	"context"
	"log"
	"time"

	"core/db"
	"core/db/queries"

	sdkapi "sdk/api"
)

type Device struct {
	db        *db.Database
	models    *Models
	id        int64
	ipaddr    string
	macaddr   string
	hostname  string
	status    sdkapi.DeviceStatus
	createdAt time.Time
}

func NewDevice(d *db.Database, m *Models) *Device {
	return &Device{db: d, models: m}
}

func BuildDevice(id int64, mac string, ip string, hostname string, status int64) *Device {
	return &Device{
		id:       id,
		ipaddr:   ip,
		macaddr:  mac,
		hostname: hostname,
		status:   sdkapi.DeviceStatus(status),
	}
}

func (self *Device) Id() int64 {
	return self.id
}

func (self *Device) Hostname() string {
	return self.hostname
}

func (self *Device) IpAddr() string {
	return self.ipaddr
}

func (self *Device) MacAddr() string {
	return self.macaddr
}

func (self *Device) Status() sdkapi.DeviceStatus {
	return self.status
}

func (self *Device) Reload(ctx context.Context) error {
	dRow, err := self.db.Queries.FindDevice(ctx, self.id)
	if err != nil {
		log.Printf("error finding device with id %v: %v", self.id, err)
	}
	self.hostname = dRow.Hostname
	self.macaddr = dRow.MacAddress
	self.ipaddr = dRow.IpAddress
	self.status = sdkapi.DeviceStatus(dRow.Status)

	return nil
}

func (self *Device) Update(ctx context.Context, params UpdateDeviceParams) error {
	err := self.db.Queries.UpdateDevice(ctx, queries.UpdateDeviceParams{
		Hostname:   params.Hostname,
		IpAddress:  params.IpAddress,
		MacAddress: params.MacAddress,
		ID:         self.id,
		Status:     int64(params.Status),
	})
	if err != nil {
		log.Printf("error updating device %v: %v", self.id, err)
		return err
	}

	self.hostname = params.Hostname
	self.ipaddr = params.IpAddress
	self.macaddr = params.MacAddress
	self.status = sdkapi.DeviceStatus(params.Status)

	return nil
}

func (self *Device) Wallet(ctx context.Context) (*Wallet, error) {
	w, err := self.db.Queries.FindWalletByDeviceId(ctx, self.id)
	if err != nil {
		log.Printf("error finding wallet by device id %v: %v", self.id, err)
		return nil, err
	}

	wallet := NewWallet(self.db, self.models)
	wallet.id = w.ID
	wallet.deviceId = w.DeviceID
	wallet.balance = w.Balance
	wallet.createdAt = w.CreatedAt

	return wallet, nil
}

func (self *Device) NextSession(ctx context.Context) (*Session, error) {
	sRow, err := self.db.Queries.FindAvailableSessionForDevice(ctx, self.id)
	if err != nil {
		log.Printf("error finding available session for device %v: %v", self.id, err)
		return nil, err
	}

	session := NewSession(self.db, self.models, &sRow)
	return session, nil
}

func (self *Device) Sessions(ctx context.Context) ([]*Session, error) {
	sessionsRow, err := self.db.Queries.FindSessionsForDev(ctx, self.id)
	if err != nil {
		log.Printf("error finding sessions for dev %v: %v", self.id, err)
		return nil, err
	}

	sessions := make([]*Session, len(sessionsRow))
	// Parse queried session rows
	for i, s := range sessionsRow {
		sessions[i] = NewSession(self.db, self.models, &s)
	}

	return sessions, nil
}

func (self *Device) Clone() *Device {
	return &Device{
		db:       self.db,
		models:   self.models,
		id:       self.id,
		ipaddr:   self.ipaddr,
		macaddr:  self.macaddr,
		hostname: self.hostname,
		status:   self.status,
	}
}
