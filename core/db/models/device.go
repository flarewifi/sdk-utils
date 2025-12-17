package models

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"core/db"
	"core/db/queries"

	sdkapi "sdk/api"
)

type Device struct {
	db        *db.Database
	models    *Models
	id        int64
	uuid      string
	ipaddr    string
	macaddr   string
	hostname  string
	status    sdkapi.DeviceStatus
	createdAt time.Time
}

func NewDevice(d *db.Database, m *Models) *Device {
	return &Device{db: d, models: m}
}

// BuildDeviceParams holds parameters for building a Device object.
type BuildDeviceParams struct {
	DB        *db.Database
	Models    *Models
	ID        int64
	UUID      string
	MacAddr   string
	IpAddr    string
	Hostname  string
	Status    sdkapi.DeviceStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BuildDevice creates a Device object from the provided parameters.
func BuildDevice(params BuildDeviceParams) *Device {
	return &Device{
		db:        params.DB,
		models:    params.Models,
		id:        params.ID,
		uuid:      params.UUID,
		ipaddr:    params.IpAddr,
		macaddr:   params.MacAddr,
		hostname:  params.Hostname,
		status:    params.Status,
		createdAt: params.CreatedAt,
	}
}

func (self *Device) ID() int64 {
	return self.id
}

func (self *Device) UUID() string {
	return self.uuid
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

// validateDeviceUpdateFields checks that required device fields are not blank
func validateDeviceUpdateFields(uuid, ip, mac string) error {
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

func (self *Device) Reload(ctx context.Context) error {
	dRow, err := self.db.Queries.FindDevice(ctx, self.id)
	if err != nil {
		log.Printf("error finding device with id %v: %v", self.id, err)
	}
	self.hostname = dRow.Hostname
	self.macaddr = dRow.MacAddress
	self.ipaddr = dRow.IpAddress
	self.uuid = dRow.Uuid
	self.status = sdkapi.DeviceStatus(dRow.Status)

	return nil
}

func (self *Device) Update(ctx context.Context, params sdkapi.UpdateDeviceParams) error {
	// Validate required fields
	if err := validateDeviceUpdateFields(params.UUID, params.Ip, params.Mac); err != nil {
		log.Printf("device validation failed: %v", err)
		return err
	}

	err := self.db.Queries.UpdateDevice(ctx, queries.UpdateDeviceParams{
		Hostname:   params.Hostname,
		IpAddress:  params.Ip,
		MacAddress: params.Mac,
		Uuid:       params.UUID,
		ID:         self.id,
		Status:     int64(params.Status),
	})
	if err != nil {
		log.Printf("error updating device %v: %v", self.id, err)
		return err
	}

	self.hostname = params.Hostname
	self.ipaddr = params.Ip
	self.macaddr = params.Mac
	self.uuid = params.UUID
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
		db:        self.db,
		models:    self.models,
		id:        self.id,
		uuid:      self.uuid,
		ipaddr:    self.ipaddr,
		macaddr:   self.macaddr,
		hostname:  self.hostname,
		status:    self.status,
		createdAt: self.createdAt,
	}
}
