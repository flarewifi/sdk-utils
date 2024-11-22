package connmgr

import (
	"context"
	"sync"
	"time"

	sdkconnmgr "sdk/api/connmgr"
)

func NewClientSession(src sdkconnmgr.ISessionSource) *ClientSession {
	s := src.Data()
	return &ClientSession{
		provider:  s.Provider,
		t:         s.Type,
		timeSecs:  s.TimeSecs,
		dataMb:    s.DataMb,
		timeCons:  s.TimeCons,
		dataCons:  s.DataCons,
		startedAt: s.StartedAt,
		expDays:   s.ExpDays,
		downMbits: s.DownMbits,
		upMbits:   s.UpMbits,
		useGlobal: s.UseGlobalSpeed,
		createdAt: s.CreatedAt,
		save:      src.Save,
		reload:    src.Reload,
	}
}

type ClientSession struct {
	mu        sync.RWMutex
	provider  string
	t         uint8
	timeSecs  uint
	dataMb    float64
	timeCons  uint
	dataCons  float64
	startedAt *time.Time
	expDays   *uint
	downMbits int
	upMbits   int
	useGlobal bool
	createdAt time.Time
	save      func(context.Context, sdkconnmgr.SessionData) error
	reload    func(context.Context) (sdkconnmgr.SessionData, error)
}

func (self *ClientSession) Provider() string {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.provider
}

func (self *ClientSession) Type() uint8 {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.t
}

func (self *ClientSession) TimeSecs() uint {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.timeSecs
}

func (self *ClientSession) DataMb() float64 {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.dataMb
}

func (self *ClientSession) TimeConsumption() uint {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.timeCons
}

func (self *ClientSession) DataConsumption() float64 {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.dataCons
}

func (self *ClientSession) RemainingTime() uint {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.timeSecs - self.timeCons
}

func (self *ClientSession) RemainingData() float64 {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.dataMb - self.dataCons
}

func (self *ClientSession) StartedAt() *time.Time {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.startedAt
}

func (self *ClientSession) ExpDays() *uint {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.expDays
}

func (self *ClientSession) ExpiresAt() *time.Time {
	self.mu.RLock()
	defer self.mu.RUnlock()

	started := self.startedAt
	exp := self.expDays

	if started != nil && exp != nil {
		exp := started.Add(time.Hour * 24 * time.Duration(*exp))
		return &exp
	}

	return nil
}

func (self *ClientSession) DownMbits() int {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.downMbits
}

func (self *ClientSession) UpMbits() int {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.upMbits
}

func (self *ClientSession) UseGlobalSpeed() bool {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.useGlobal
}

func (self *ClientSession) CreatedAt() time.Time {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.createdAt
}

func (self *ClientSession) IncTimeCons(sec uint) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.timeCons += sec
}

func (self *ClientSession) IncDataCons(mbytes float64) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.dataCons += mbytes
}

func (self *ClientSession) SetTimeSecs(sec uint) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.timeSecs = sec
}

func (self *ClientSession) SetDataMb(mbytes float64) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.dataMb = mbytes
}

func (self *ClientSession) SetTimeCons(sec uint) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.timeCons = sec
}

func (self *ClientSession) SetDataCons(mbytes float64) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.dataCons = mbytes
}

func (self *ClientSession) SetStartedAt(started *time.Time) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.startedAt = started
}

func (self *ClientSession) SetExpDays(exp *uint) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.expDays = exp
}

func (self *ClientSession) SetDownMbits(mbits int) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.downMbits = mbits
}

func (self *ClientSession) SetUpMbits(mbits int) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.upMbits = mbits
}

func (self *ClientSession) SetUseGlobalSpeed(useGlobal bool) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.useGlobal = useGlobal
}

func (self *ClientSession) Save(ctx context.Context) error {
	self.mu.RLock()
	defer self.mu.RUnlock()

	data := sdkconnmgr.SessionData{
		Provider:       self.provider,
		Type:           self.t,
		TimeSecs:       self.timeSecs,
		DataMb:         self.dataMb,
		TimeCons:       self.timeCons,
		DataCons:       self.dataCons,
		StartedAt:      self.startedAt,
		ExpDays:        self.expDays,
		DownMbits:      self.downMbits,
		UpMbits:        self.upMbits,
		UseGlobalSpeed: self.useGlobal,
		CreatedAt:      self.createdAt,
	}

	return self.save(ctx, data)
}

func (self *ClientSession) Reload(ctx context.Context) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	s, err := self.reload(ctx)
	if err != nil {
		return err
	}

	self.provider = s.Provider
	self.t = s.Type
	self.timeSecs = s.TimeSecs
	self.dataMb = s.DataMb
	self.timeCons = s.TimeCons
	self.dataCons = s.DataCons
	self.startedAt = s.StartedAt
	self.expDays = s.ExpDays
	self.downMbits = s.DownMbits
	self.upMbits = s.UpMbits
	self.useGlobal = s.UseGlobalSpeed

	return nil
}
