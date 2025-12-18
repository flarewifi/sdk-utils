package sessmgr

import (
	"context"
	"fmt"
	"sync"

	"core/db"
	"core/db/models"
	"core/internal/utils/events"
	"core/tools/sse"

	sdkapi "sdk/api"
)

type ClientDevice struct {
	mu       sync.RWMutex
	db       *db.Database
	mdls     *models.Models
	id       int64
	uuid     string
	mac      string
	ip       string
	hostname string
	status   sdkapi.DeviceStatus
}

func NewClientDevice(dtb *db.Database, mdls *models.Models, d *models.Device) *ClientDevice {
	return &ClientDevice{
		db:       dtb,
		mdls:     mdls,
		id:       d.ID(),
		uuid:     d.UUID(),
		mac:      d.MacAddr(),
		ip:       d.IpAddr(),
		hostname: d.Hostname(),
		status:   d.Status(),
	}
}

func (self *ClientDevice) ID() int64 {
	self.mu.RLock()
	defer self.mu.RUnlock()
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

func (self *ClientDevice) Update(ctx context.Context, params sdkapi.UpdateDeviceParams) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	err := self.mdls.Device().Update(ctx, models.UpdateDeviceParams{
		ID:         self.id,
		MacAddress: params.Mac,
		IpAddress:  params.Ip,
		Hostname:   params.Hostname,
		UUID:       params.UUID,
		Status:     params.Status,
	})
	if err != nil {
		return err
	}

	self.hostname = params.Hostname
	self.mac = params.Mac
	self.ip = params.Ip
	self.uuid = params.UUID
	self.status = sdkapi.DeviceStatus(params.Status)

	return nil
}

func (self *ClientDevice) Emit(event string, data []byte) {
	channel := self.GetEventChannel(event)
	sse.Emit(self.MacAddr(), event, data)
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
	return fmt.Sprintf("%s:%s", self.MacAddr(), event)
}
