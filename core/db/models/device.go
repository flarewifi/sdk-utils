package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"core/db"
	"core/db/queries"

	sdkapi "sdk/api"
)

type Device struct {
	db          *db.Database
	models      *Models
	id          int64
	uuid        string
	cookieToken string
	ipv4addr    string
	ipv6addr    string
	macaddr     string
	hostname    string
	status      sdkapi.DeviceStatus
	createdAt   time.Time
	updatedAt   time.Time
}

func NewDevice(d *db.Database, m *Models) *Device {
	return &Device{db: d, models: m}
}

// BuildDeviceParams holds parameters for building a Device object.
type BuildDeviceParams struct {
	DB          *db.Database
	Models      *Models
	ID          int64
	UUID        string
	CookieToken string
	MacAddr     string
	Ipv4Addr    string
	Ipv6Addr    string
	Hostname    string
	Status      sdkapi.DeviceStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// BuildDevice creates a Device object from the provided parameters.
func BuildDevice(params BuildDeviceParams) *Device {
	return &Device{
		db:          params.DB,
		models:      params.Models,
		id:          params.ID,
		uuid:        params.UUID,
		cookieToken: params.CookieToken,
		ipv4addr:    params.Ipv4Addr,
		ipv6addr:    params.Ipv6Addr,
		macaddr:     params.MacAddr,
		hostname:    params.Hostname,
		status:      params.Status,
		createdAt:   params.CreatedAt,
		updatedAt:   params.UpdatedAt,
	}
}

func (self *Device) ID() int64 {
	return self.id
}

func (self *Device) UUID() string {
	return self.uuid
}

func (self *Device) CookieToken() string {
	return self.cookieToken
}

func (self *Device) Hostname() string {
	return self.hostname
}

// Ipv4Addr returns the device's IPv4 address (empty if not available).
func (self *Device) Ipv4Addr() string {
	return self.ipv4addr
}

// Ipv6Addr returns the device's IPv6 address (empty if not available).
func (self *Device) Ipv6Addr() string {
	return self.ipv6addr
}

// IpAddr returns the primary IP address for backward compatibility.
// Returns IPv4 if available, otherwise IPv6.
func (self *Device) IpAddr() string {
	if self.ipv4addr != "" {
		return self.ipv4addr
	}
	return self.ipv6addr
}

func (self *Device) MacAddr() string {
	return self.macaddr
}

func (self *Device) Status() sdkapi.DeviceStatus {
	return self.status
}

func (self *Device) CreatedAt() time.Time {
	return self.createdAt
}

func (self *Device) UpdatedAt() time.Time {
	return self.updatedAt
}

// validateDeviceUpdateFields checks that required device fields are not blank
func validateDeviceUpdateFields(uuid, ipv4, ipv6, mac string) error {
	if strings.TrimSpace(uuid) == "" {
		return fmt.Errorf("uuid cannot be blank")
	}
	if strings.TrimSpace(ipv4) == "" && strings.TrimSpace(ipv6) == "" {
		return fmt.Errorf("at least one IP address (IPv4 or IPv6) is required")
	}
	if strings.TrimSpace(mac) == "" {
		return fmt.Errorf("mac address cannot be blank")
	}
	return nil
}

func (self *Device) Reload(ctx context.Context) error {
	dRow, err := self.db.Queries.FindDevice(ctx, self.id)
	if err != nil {
		return err
	}

	self.hostname = dRow.Hostname
	self.macaddr = dRow.MacAddress // comes from JOIN
	self.ipv4addr = dRow.Ipv4Addr
	self.ipv6addr = dRow.Ipv6Addr
	self.uuid = dRow.Uuid
	self.cookieToken = dRow.CookieToken
	self.status = sdkapi.DeviceStatus(dRow.Status)

	return nil
}

func (self *Device) Update(ctx context.Context, params sdkapi.UpdateDeviceParams) error {
	// Validate required fields
	if err := validateDeviceUpdateFields(params.UUID, params.Ipv4, params.Ipv6, params.Mac); err != nil {
		return err
	}

	// Update device record (without MAC address)
	err := self.db.Queries.UpdateDevice(ctx, queries.UpdateDeviceParams{
		Hostname: params.Hostname,
		Ipv4Addr: params.Ipv4,
		Ipv6Addr: params.Ipv6,
		Uuid:     params.UUID,
		ID:       self.id,
		Status:   int64(params.Status),
	})
	if err != nil {
		return err
	}

	// Always record MAC address to update last_seen_at and ensure is_current is set
	// RecordMacAddress is idempotent - if MAC exists, it just updates timestamp
	if err := self.models.DeviceMac().RecordMacAddress(ctx, self.id, params.Mac); err != nil {
		return fmt.Errorf("failed to record MAC address for device %d: %w", self.id, err)
	}

	self.hostname = params.Hostname
	self.ipv4addr = params.Ipv4
	self.ipv6addr = params.Ipv6
	self.macaddr = params.Mac
	self.uuid = params.UUID
	self.status = sdkapi.DeviceStatus(params.Status)

	return nil
}

func (self *Device) Wallet(ctx context.Context) (*Wallet, error) {
	w, err := self.db.Queries.FindWalletByDeviceId(ctx, self.id)
	if err != nil {
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
		return nil, err
	}

	session := NewSession(self.db, self.models, &sRow)
	return session, nil
}

func (self *Device) Sessions(ctx context.Context) ([]*Session, error) {
	sessionsRow, err := self.db.Queries.FindSessionsForDev(ctx, self.id)
	if err != nil {
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
		db:          self.db,
		models:      self.models,
		id:          self.id,
		uuid:        self.uuid,
		cookieToken: self.cookieToken,
		ipv4addr:    self.ipv4addr,
		ipv6addr:    self.ipv6addr,
		macaddr:     self.macaddr,
		hostname:    self.hostname,
		status:      self.status,
		createdAt:   self.createdAt,
		updatedAt:   self.updatedAt,
	}
}
