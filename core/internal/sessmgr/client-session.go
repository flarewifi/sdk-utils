package sessmgr

import (
	"context"
	"sync"
	"time"

	"core/db"
	"core/db/models"
	sdkapi "sdk/api"
)

func NewClientSession(dtb *db.Database, mdls *models.Models, pluginsMgr sdkapi.IPluginsMgrApi, s *models.Session) *ClientSession {
	cs := &ClientSession{db: dtb, mdls: mdls, pluginsMgr: pluginsMgr}
	cs.load(s)
	return cs
}

type ClientSession struct {
	mu          sync.RWMutex
	db          *db.Database
	mdls        *models.Models
	pluginsMgr  sdkapi.IPluginsMgrApi
	id          int64
	uuid        string
	providerPkg string
	devId       int64
	sessionType string
	timeSecs    int
	dataMb      float64
	timeCons    int
	dataCons    float64
	startedAt   *time.Time
	resumedAt   *time.Time
	expDays     *int
	downMbits   int
	upMbits     int
	useGlobal   bool
	createdAt   time.Time
	updatedAt   time.Time
}

func (self *ClientSession) Save(ctx context.Context) error {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.mdls.Session().Update(ctx, models.UpdateSessionParams{
		ID:          self.id,
		UUID:        self.uuid,
		ProviderPkg: self.providerPkg,
		DeviceID:    self.devId,
		SessionType: sdkapi.SessionType(self.sessionType),
		TimeSecs:    self.timeSecs,
		DataMbytes:  self.dataMb,
		TimeCons:    self.timeCons,
		DataCons:    self.dataCons,
		StartedAt:   self.startedAt,
		ResumedAt:   self.resumedAt,
		ExpDays:     self.expDays,
		DownMbits:   self.downMbits,
		UpMbits:     self.upMbits,
		UseGlobal:   self.useGlobal,
	})
}

func (self *ClientSession) Reload(ctx context.Context) (err error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	s, err := self.mdls.Session().Find(ctx, self.id)

	if err != nil {
		return err
	}

	self.load(s)
	return nil
}

func (self *ClientSession) load(s *models.Session) {
	self.id = s.ID()
	self.uuid = s.UUID()
	self.providerPkg = s.ProviderPkg()
	self.devId = s.DeviceID()
	self.sessionType = s.SessionType()
	self.timeSecs = s.TimeSecs()
	self.dataMb = s.DataMbyte()
	self.timeCons = s.TimeConsumed()
	self.dataCons = s.DataConsumed()
	self.downMbits = s.DownMbits()
	self.upMbits = s.UpMbits()
	self.useGlobal = s.UseGlobal()
	self.expDays = s.ExpDays()
	self.startedAt = s.StartedAt()
	self.resumedAt = s.ResumedAt()
	self.createdAt = s.CreatedAt()
	self.updatedAt = s.UpdatedAt()
}

// ID returns the session's ID.
func (self *ClientSession) ID() int64 {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.id
}

// UUID returns the session's UUID.
func (self *ClientSession) UUID() string {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.uuid
}

// DeviceID returns the device ID that owns this session.
func (self *ClientSession) DeviceID() int64 {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.devId
}

// Plugin returns the provider plugin of the session record.
func (self *ClientSession) Plugin() sdkapi.IPluginApi {
	self.mu.RLock()
	defer self.mu.RUnlock()
	if plugin, found := self.pluginsMgr.FindByPkg(self.providerPkg); found {
		return plugin
	}
	return nil
}

// Type returns the session type.
func (self *ClientSession) Type() sdkapi.SessionType {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return sdkapi.SessionType(self.sessionType)
}

// TimeSecs returns the session's available time in seconds.
func (self *ClientSession) TimeSecs() (sec int) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.timeSecs
}

// DataMb returns the session's available data in megabytes.
func (self *ClientSession) DataMb() (mbytes float64) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.dataMb
}

// TimeConsumption returns the session's time consumption in seconds.
// If session is currently running (resumed_at != nil), includes elapsed time since resumed_at.
func (self *ClientSession) TimeConsumption() (sec int) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	consumption := self.timeCons

	// If session is running, add elapsed time since resumed_at
	if self.resumedAt != nil {
		elapsed := int(time.Since(*self.resumedAt).Seconds())
		consumption += elapsed
	}

	return consumption
}

// DataConsumption returns the session's data consumption in megabytes.
// Note: Data consumption is tracked in real-time via traffic monitoring,
// so this returns the saved value without additional elapsed time calculation.
func (self *ClientSession) DataConsumption() (mbytes float64) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.dataCons
}

// RemainingTime returns the session's remaining time in seconds.
func (self *ClientSession) RemainingTime() (sec int) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	remaining := self.timeSecs - self.timeCons

	// If session is running, subtract elapsed time since resumed_at
	if self.resumedAt != nil {
		elapsed := int(time.Since(*self.resumedAt).Seconds())
		remaining -= elapsed
	}

	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

// RemainingData returns the session's remaining data in megabytes.
func (self *ClientSession) RemainingData() (mbytes float64) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.dataMb - self.dataCons
}

// StartedAt returns the time when session was first started.
func (self *ClientSession) StartedAt() *time.Time {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.startedAt
}

// ResumedAt returns the time when session was last resumed.
func (self *ClientSession) ResumedAt() *time.Time {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.resumedAt
}

// CreatedAt returns the created at time.
func (self *ClientSession) CreatedAt() time.Time {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.createdAt
}

// UpdatedAt returns the updated at time.
func (self *ClientSession) UpdatedAt() time.Time {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.updatedAt
}

// ExpDays returns the session's expiration time in days.
// If session has no expiration, it returns nil.
func (self *ClientSession) ExpDays() *int {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.expDays
}

// ExpiresAt returns the time when session will expire.
// If session has no expiration, it returns nil.
// Expiration time is calculated from the time when session was started.
func (self *ClientSession) ExpiresAt() *time.Time {
	self.mu.RLock()
	defer self.mu.RUnlock()
	if self.startedAt != nil && self.expDays != nil {
		exp := self.startedAt.Add(time.Hour * 24 * time.Duration(*self.expDays))
		return &exp
	}
	return nil
}

// DownMbits returns the session's download speed limit in megabits per second.
func (self *ClientSession) DownMbits() int {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.downMbits
}

// UpMbits returns the session's upload speed limit in megabits per second.
func (self *ClientSession) UpMbits() int {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.upMbits
}

// UseGlobalSpeed returns whether session uses global speed limits.
func (self *ClientSession) UseGlobalSpeed() bool {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.useGlobal
}

// IncTimeCons increases the session's time consumption in seconds.
// This value is not saved until Save() method is called.
func (self *ClientSession) IncTimeCons(sec int) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.timeCons += sec
}

// IncDataCons increases the session's data consumption in megabytes.
// This value is not saved until Save() method is called.
func (self *ClientSession) IncDataCons(mbytes float64) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.dataCons += mbytes
}

// SetTimeSecs sets the session's available time in seconds.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetTimeSecs(sec int) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.timeSecs = sec
}

// SetDataMb sets the session's available data in megabytes.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetDataMb(mbytes float64) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.dataMb = mbytes
}

// SetTimeCons sets the session's time consumption in seconds.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetTimeCons(sec int) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.timeCons = sec
}

// SetDataCons sets the session's data consumption in megabytes.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetDataCons(mbytes float64) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.dataCons = mbytes
}

// SetStartedAt sets the time when session was first started.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetStartedAt(started *time.Time) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.startedAt = started
}

// SetResumedAt sets the time when session was last resumed.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetResumedAt(resumed *time.Time) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.resumedAt = resumed
}

// SetExpDays sets the session's expiration time in days.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetExpDays(exp *int) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.expDays = exp
}

// SetDownMbits sets the session's download speed limit in megabits per second.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetDownMbits(mbits int) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.downMbits = mbits
}

// SetUpMbits sets the session's upload speed limit in megabits per second.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetUpMbits(mbits int) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.upMbits = mbits
}

// SetUseGlobalSpeed sets whether session uses global speed limits.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetUseGlobalSpeed(useGlobal bool) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.useGlobal = useGlobal
}
