package models

import (
	"context"
	"log"
	"time"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Device struct {
	db        *db.Database
	models    *Models
	id        pgtype.UUID
	ipaddr    string
	macaddr   string
	hostname  string
	createdAt time.Time
}

func NewDevice(d *db.Database, m *Models) *Device {
	return &Device{db: d, models: m}
}

func BuildDevice(id pgtype.UUID, mac string, ip string, hostname string) *Device {
	return &Device{
		id:       id,
		ipaddr:   ip,
		macaddr:  mac,
		hostname: hostname,
	}
}

func (self *Device) Id() pgtype.UUID {
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

func (self *Device) Reload(tx pgx.Tx, ctx context.Context) error {
	qtx := self.db.Queries.WithTx(tx)
	dRow, err := qtx.FindDevice(ctx, self.id)
	if err != nil {
		log.Printf("error finding device with id %v: %v", self.id, err)
	}
	self.hostname = dRow.Hostname
	self.macaddr = dRow.IpAddress
	self.ipaddr = dRow.MacAddress

	return nil
}

func (self *Device) Update(tx pgx.Tx, ctx context.Context, mac string, ip string, hostname string) error {
	qtx := self.db.Queries.WithTx(tx)
	err := qtx.UpdateDevice(ctx, queries.UpdateDeviceParams{
		Hostname:   hostname,
		IpAddress:  ip,
		MacAddress: mac,
		ID:         self.id,
	})
	if err != nil {
		log.Printf("error updating device %v: %v", self.id, err)
		return err
	}

	self.hostname = hostname
	self.ipaddr = ip
	self.macaddr = mac

	return nil
}

func (self *Device) Wallet(tx pgx.Tx, ctx context.Context) (*Wallet, error) {
	qtx := self.db.Queries.WithTx(tx)
	w, err := qtx.FindWalletByDeviceId(ctx, self.id)
	if err != nil {
		log.Printf("error finding wallet by device id %v: %v", self.id, err)
		return nil, err
	}

	wallet := NewWallet(self.db, self.models)
	wallet.id = w.ID
	wallet.deviceId = w.DeviceID
	wallet.balance = sdkutils.PgNumericToFloat64(w.Balance)
	wallet.createdAt = w.CreatedAt.Time

	return wallet, nil
}

func (self *Device) NextSession(tx pgx.Tx, ctx context.Context) (*Session, error) {
	qtx := self.db.Queries.WithTx(tx)
	sRow, err := qtx.FindAvailableSessionForDevice(ctx, self.id)
	if err != nil {
		log.Printf("error finding available session for device %v: %v", self.id, err)
		return nil, err
	}

	session := NewSession(self.db, self.models, &sRow)
	return session, nil
}

func (self *Device) Sessions(tx pgx.Tx, ctx context.Context) ([]*Session, error) {
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
	}
}
