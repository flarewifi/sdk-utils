package models

import (
	"context"
	"database/sql"
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

func (self *Device) Reload(tx *sql.Tx, ctx context.Context) error {
	qtx := self.db.Queries.WithTx(tx)
	dRow, err := qtx.FindDevice(ctx, self.id)
	if err != nil {
		log.Printf("error finding device with id %v: %v", self.id, err)
	}
	self.hostname = dRow.Hostname
	self.macaddr = dRow.MacAddress
	self.ipaddr = dRow.IpAddress
	self.status = sdkapi.DeviceStatus(dRow.Status)

	return nil
}

func (self *Device) Update(tx *sql.Tx, ctx context.Context, mac string, ip string, hostname string, status int) error {
	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdateDevice(ctx, queries.UpdateDeviceParams{
		Hostname:   hostname,
		IpAddress:  ip,
		MacAddress: mac,
		ID:         self.id,
		Status:     int64(status),
	})
	if err != nil {
		log.Printf("error updating device %v: %v", self.id, err)
		return err
	}

	self.hostname = hostname
	self.ipaddr = ip
	self.macaddr = mac
	self.status = sdkapi.DeviceStatus(status)

	return nil
}

func (self *Device) Wallet(tx *sql.Tx, ctx context.Context) (*Wallet, error) {
	qtx := self.db.Queries.WithTx(tx)
	w, err := qtx.FindWalletByDeviceId(ctx, self.id)
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

func (self *Device) NextSession(tx *sql.Tx, ctx context.Context) (*Session, error) {
	qtx := self.db.Queries.WithTx(tx)
	sRow, err := qtx.FindAvailableSessionForDevice(ctx, self.id)
	if err != nil {
		log.Printf("error finding available session for device %v: %v", self.id, err)
		return nil, err
	}

	session := NewSession(self.db, self.models, &sRow)
	return session, nil
}

func (self *Device) Sessions(tx *sql.Tx, ctx context.Context) ([]*Session, error) {
	qtx := self.db.Queries.WithTx(tx)
	sessionsRow, err := qtx.FindSessionsForDev(ctx, self.id)
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
