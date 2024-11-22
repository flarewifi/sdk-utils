package models

import (
	"context"
	"log"
	"time"

	"core/internal/db"
	"core/internal/db/sqlc"
	"core/internal/utils/pg"

	"github.com/jackc/pgx/v5/pgtype"
)

type Device struct {
	db        *db.Database
	models    *Models
	id        pgtype.UUID
	macAddr   string
	ipAddr    string
	hostname  string
	createdAt time.Time
}

func NewDevice(d *db.Database, m *Models) *Device {
	return &Device{db: d, models: m}
}

func BuildDevice(id pgtype.UUID, mac string, ip string, hostname string) *Device {
	return &Device{
		id:       id,
		macAddr:  mac,
		ipAddr:   ip,
		hostname: hostname,
	}
}

func (self *Device) Id() pgtype.UUID {
	return self.id
}

func (self *Device) Hostname() string {
	return self.hostname
}

func (self *Device) IpAddress() string {
	return self.ipAddr
}

func (self *Device) MacAddress() string {
	return self.macAddr
}

func (self *Device) Reload(ctx context.Context) error {
	dRow, err := self.db.Queries.FindDevice(ctx, self.id)
	if err != nil {
		log.Printf("error finding device with id %v: %v", self.id, err)
	}
	self.hostname = dRow.Hostname.String
	self.ipAddr = dRow.IpAddress
	self.macAddr = dRow.MacAddress

	return nil
}

func (self *Device) Update(ctx context.Context, mac string, ip string, hostname string) error {
	err := self.db.Queries.UpdateDevice(ctx, sqlc.UpdateDeviceParams{
		Hostname:   pgtype.Text{String: hostname, Valid: hostname != ""},
		IpAddress:  ip,
		MacAddress: mac,
		ID:         self.id,
	})
	if err != nil {
		log.Printf("error updating device %v: %v", self.id, err)
		return err
	}

	self.hostname = hostname
	self.macAddr = mac
	self.ipAddr = ip

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
	wallet.balance = pg.NumericToFloat64(w.Balance)
	wallet.createdAt = w.CreatedAt.Time

	return wallet, nil
}

func (self *Device) NextSession(ctx context.Context) (*Session, error) {
	sRow, err := self.db.Queries.FindAvlSessionForDev(ctx, self.id)
	if err != nil {
		log.Printf("error finding available session for device %v: %v", self.id, err)
		return nil, err
	}

	expDaysUint := uint(sRow.ExpDays.Int32)
	expDays := &expDaysUint

	session := &Session{
		db:          self.db,
		models:      self.models,
		id:          sRow.ID,
		deviceId:    sRow.DeviceID,
		sessionType: uint8(sRow.SessionType),
		timeSecs:    uint(sRow.TimeSecs.Int32),
		dataMb:      pg.NumericToFloat64(sRow.DataMbytes),
		timeCons:    uint(sRow.ConsumptionSecs.Int32),
		dataCons:    pg.NumericToFloat64(sRow.ConsumptionMb),
		startedAt:   &sRow.StartedAt.Time,
		expDays:     expDays,
		// TODO: find out the proper calculation of this field
		// expiresAt:   sRow.ExpiresAt,
		downMbits: int(sRow.DownMbits),
		upMbits:   int(sRow.DownMbits),
		useGlobal: sRow.UseGlobal,
		createdAt: sRow.CreatedAt.Time,
	}

	return session, nil
}

func (self *Device) Sessions(ctx context.Context) ([]*Session, error) {
	var sessions []*Session

	sessionsRow, err := self.db.Queries.FindSessionsForDev(ctx, self.id)
	if err != nil {
		log.Printf("error finding sessions for dev %v: %v", self.id, err)
		return nil, err
	}

	// Parse queried session rows
	for _, s := range sessionsRow {
		expDays := uint(s.ExpDays.Int32)

		sessions = append(sessions, &Session{
			db:          self.db,
			models:      self.models,
			id:          s.ID,
			deviceId:    s.DeviceID,
			sessionType: uint8(s.SessionType),
			timeSecs:    uint(s.TimeSecs.Int32),
			dataMb:      pg.NumericToFloat64(s.DataMbytes),
			timeCons:    uint(s.ConsumptionSecs.Int32),
			dataCons:    pg.NumericToFloat64(s.ConsumptionMb),
			startedAt:   &s.StartedAt.Time,
			expDays:     &expDays,
			// TODO: find out the proper calculation of this field
			// expiresAt: s.ExpiresAt,
			downMbits: int(s.DownMbits),
			upMbits:   int(s.UpMbits),
			useGlobal: s.UseGlobal,
			createdAt: s.CreatedAt.Time,
		})
	}

	return sessions, nil
}

func (self *Device) Clone() *Device {
	return &Device{
		db:       self.db,
		models:   self.models,
		id:       self.id,
		macAddr:  self.macAddr,
		ipAddr:   self.ipAddr,
		hostname: self.hostname,
	}
}
