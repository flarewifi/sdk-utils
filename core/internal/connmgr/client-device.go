package connmgr

import (
	"context"
	"fmt"
	"sync"

	"core/internal/db"
	"core/internal/db/models"
	"core/internal/utils/events"
	"core/internal/utils/sse"

	"github.com/jackc/pgx/v5/pgtype"
)

type ClientDevice struct {
	mu       sync.RWMutex
	db       *db.Database
	mdls     *models.Models
	id       pgtype.UUID
	mac      string
	ip       string
	hostname string
}

func NewClientDevice(dtb *db.Database, mdls *models.Models, d *models.Device) *ClientDevice {
	return &ClientDevice{
		db:       dtb,
		mdls:     mdls,
		id:       d.Id(),
		mac:      d.MacAddress(),
		ip:       d.IpAddress(),
		hostname: d.Hostname(),
	}
}

func (self *ClientDevice) Id() pgtype.UUID {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.id
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

func (self *ClientDevice) Update(ctx context.Context, mac string, ip string, hostname string) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	err := self.mdls.Device().Update(ctx, self.id, self.mac, self.ip, self.hostname)
	if err != nil {
		return err
	}

	self.hostname = hostname
	self.mac = mac
	self.ip = ip

	return nil
}

func (self *ClientDevice) Emit(event string, data interface{}) {
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
