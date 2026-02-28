package sessmgr

import (
	"context"
	"fmt"
	"sync"
	"time"

	"core/db"
	"core/db/models"
	"core/internal/modules/events"
	"core/utils/sse"

	sdkapi "sdk/api"
)

type ClientDevice struct {
	// === IMMUTABLE after creation (no lock needed) ===
	db        *db.Database
	mdls      *models.Models
	id        int64
	createdAt time.Time

	// === MUTABLE (protected by mu) ===
	mu        sync.RWMutex
	uuid      string
	mac       string
	ip        string
	hostname  string
	status    sdkapi.DeviceStatus
	updatedAt time.Time
}

func NewClientDevice(dtb *db.Database, mdls *models.Models, d *models.Device) *ClientDevice {
	return &ClientDevice{
		db:        dtb,
		mdls:      mdls,
		id:        d.ID(),
		createdAt: d.CreatedAt(),
		uuid:      d.UUID(),
		mac:       d.MacAddr(),
		ip:        d.IpAddr(),
		hostname:  d.Hostname(),
		status:    d.Status(),
		updatedAt: d.UpdatedAt(),
	}
}

// ID returns the device's database ID (immutable, no lock needed).
func (self *ClientDevice) ID() int64 {
	return self.id
}

func (self *ClientDevice) UUID() string {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.uuid
}

func (self *ClientDevice) Hostname() string {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.hostname
}

func (self *ClientDevice) MacAddr() string {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.mac
}

func (self *ClientDevice) IpAddr() string {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.ip
}

func (self *ClientDevice) Status() sdkapi.DeviceStatus {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.status
}

// CreatedAt returns the device's creation timestamp (immutable, no lock needed).
func (self *ClientDevice) CreatedAt() time.Time {
	return self.createdAt
}

// UpdatedAt returns the device's last update timestamp.
func (self *ClientDevice) UpdatedAt() time.Time {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.updatedAt
}

// Data returns a snapshot of all device data fields.
// This method acquires the mutex once and returns all fields,
// reducing lock contention compared to calling individual getters.
func (self *ClientDevice) Data() sdkapi.DeviceData {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return sdkapi.DeviceData{
		ID:        self.id,
		UUID:      self.uuid,
		MacAddr:   self.mac,
		IpAddr:    self.ip,
		Hostname:  self.hostname,
		Status:    self.status,
		CreatedAt: self.createdAt,
		UpdatedAt: self.updatedAt,
	}
}

func (self *ClientDevice) Update(ctx context.Context, params sdkapi.UpdateDeviceParams) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	// If a field is not provided (zero value), keep the existing value.
	if params.Status == 0 {
		params.Status = self.status
	}

	err := self.mdls.Device().Update(ctx, models.UpdateDeviceParams{
		ID:         self.id,
		MacAddress: params.Mac,
		IpAddress:  params.Ip,
		Hostname:   params.Hostname,
		UUID:       params.UUID,
		Status:     int(params.Status),
	})
	if err != nil {
		return err
	}

	self.hostname = params.Hostname
	self.mac = params.Mac
	self.ip = params.Ip
	self.uuid = params.UUID
	self.status = sdkapi.DeviceStatus(params.Status)
	self.updatedAt = time.Now()

	return nil
}

func (self *ClientDevice) Emit(event string, data []byte) {
	channel := self.GetEventChannel(event)
	sse.Emit(fmt.Sprintf("%d", self.ID()), event, data)
	events.Emit(channel, data)
}

func (self *ClientDevice) Subscribe(event string) <-chan []byte {
	channel := self.GetEventChannel(event)
	ch := events.Subscribe(channel)
	return ch
}

func (self *ClientDevice) Unsubscribe(event string, ch <-chan []byte) {
	channel := self.GetEventChannel(event)
	events.Unsubscribe(channel, ch)
}

func (self *ClientDevice) GetEventChannel(event string) string {
	return fmt.Sprintf("%d:%s", self.ID(), event)
}
