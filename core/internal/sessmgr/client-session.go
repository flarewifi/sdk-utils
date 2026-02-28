package sessmgr

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"core/db"
	"core/db/models"
	sdkapi "sdk/api"
)

// sessionData holds all session fields as an immutable snapshot.
// This enables lock-free reads via atomic.Pointer.
type sessionData struct {
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

	// Dirty tracking - which fields changed since last save/load
	dirtyTime      bool // timeSecs or timeCons changed
	dirtyData      bool // dataMb or dataCons changed
	dirtyBandwidth bool // downMbits, upMbits, or useGlobal changed
}

// copyTimePtr creates a deep copy of a time pointer to avoid shared state
func copyTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	copied := *t
	return &copied
}

// copyIntPtr creates a deep copy of an int pointer to avoid shared state
func copyIntPtr(i *int) *int {
	if i == nil {
		return nil
	}
	copied := *i
	return &copied
}

// copy creates a deep copy of sessionData for modification
func (d *sessionData) copy() sessionData {
	return sessionData{
		id:          d.id,
		uuid:        d.uuid,
		providerPkg: d.providerPkg,
		devId:       d.devId,
		sessionType: d.sessionType,
		timeSecs:    d.timeSecs,
		dataMb:      d.dataMb,
		timeCons:    d.timeCons,
		dataCons:    d.dataCons,
		startedAt:   copyTimePtr(d.startedAt),
		resumedAt:   copyTimePtr(d.resumedAt),
		expDays:     copyIntPtr(d.expDays),
		downMbits:   d.downMbits,
		upMbits:     d.upMbits,
		useGlobal:   d.useGlobal,
		createdAt:   d.createdAt,
		updatedAt:   d.updatedAt,
		// Preserve dirty flags during copy
		dirtyTime:      d.dirtyTime,
		dirtyData:      d.dirtyData,
		dirtyBandwidth: d.dirtyBandwidth,
	}
}

func NewClientSession(dtb *db.Database, mdls *models.Models, pluginsMgr sdkapi.IPluginsMgrApi, s *models.Session) *ClientSession {
	cs := &ClientSession{
		db:         dtb,
		mdls:       mdls,
		pluginsMgr: pluginsMgr,
	}
	cs.loadFromModel(s)
	return cs
}

// SetOnSave sets the callback function for session save events.
func (self *ClientSession) SetOnSave(fn sdkapi.SessionSaveCallback) {
	self.onSave = fn
}

// ClientSession wraps session data with lock-free reads and synchronized writes.
// Reads use atomic.Pointer for zero-lock access.
// Writes use copy-modify-swap pattern protected by writeMu.
type ClientSession struct {
	// Dependencies - immutable after creation, no lock needed
	db         *db.Database
	mdls       *models.Models
	pluginsMgr sdkapi.IPluginsMgrApi

	// Callback for save notifications (set by SessionsMgr)
	// Called after Save() to apply side effects to running sessions
	onSave sdkapi.SessionSaveCallback

	// Session data - atomic pointer for lock-free reads
	data atomic.Pointer[sessionData]

	// Write mutex - only needed for modifications (copy-modify-swap)
	writeMu sync.Mutex
}

func (self *ClientSession) Save(ctx context.Context) error {
	// Take a consistent snapshot and clear dirty flags atomically.
	// This ensures no writes are lost between reading flags and clearing them.
	self.writeMu.Lock()
	d := self.data.Load()

	// Collect changed fields from the current snapshot
	changedFields := sdkapi.SessionChangedFields{
		Time:      d.dirtyTime,
		Data:      d.dirtyData,
		Bandwidth: d.dirtyBandwidth,
	}

	// Clear dirty flags immediately so concurrent setters mark new changes
	if changedFields.Time || changedFields.Data || changedFields.Bandwidth {
		newData := d.copy()
		newData.dirtyTime = false
		newData.dirtyData = false
		newData.dirtyBandwidth = false
		self.data.Store(&newData)
	}
	self.writeMu.Unlock()

	// Save to database (outside lock - DB operations can be slow)
	err := self.mdls.Session().Update(ctx, models.UpdateSessionParams{
		ID:          d.id,
		UUID:        d.uuid,
		ProviderPkg: d.providerPkg,
		DeviceID:    d.devId,
		SessionType: sdkapi.SessionType(d.sessionType),
		TimeSecs:    d.timeSecs,
		DataMbytes:  d.dataMb,
		TimeCons:    d.timeCons,
		DataCons:    d.dataCons,
		StartedAt:   d.startedAt,
		ResumedAt:   d.resumedAt,
		ExpDays:     d.expDays,
		DownMbits:   d.downMbits,
		UpMbits:     d.upMbits,
		UseGlobal:   d.useGlobal,
	})
	if err != nil {
		return err
	}

	// Notify callback if any fields changed and callback is set
	if self.onSave != nil && (changedFields.Time || changedFields.Data || changedFields.Bandwidth) {
		return self.onSave(sdkapi.SessionSaveParams{
			Ctx:           ctx,
			Session:       self,
			ChangedFields: changedFields,
		})
	}

	return nil
}

func (self *ClientSession) Reload(ctx context.Context) (err error) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	d := self.data.Load()
	s, err := self.mdls.Session().Find(ctx, d.id)
	if err != nil {
		return err
	}

	self.loadFromModel(s)
	return nil
}

// loadFromModel creates a new sessionData snapshot from a models.Session
func (self *ClientSession) loadFromModel(s *models.Session) {
	newData := &sessionData{
		id:          s.ID(),
		uuid:        s.UUID(),
		providerPkg: s.ProviderPkg(),
		devId:       s.DeviceID(),
		sessionType: s.SessionType(),
		timeSecs:    s.TimeSecs(),
		dataMb:      s.DataMbyte(),
		timeCons:    s.TimeConsumed(),
		dataCons:    s.DataConsumed(),
		downMbits:   s.DownMbits(),
		upMbits:     s.UpMbits(),
		useGlobal:   s.UseGlobal(),
		expDays:     copyIntPtr(s.ExpDays()),
		startedAt:   copyTimePtr(s.StartedAt()),
		resumedAt:   copyTimePtr(s.ResumedAt()),
		createdAt:   s.CreatedAt(),
		updatedAt:   s.UpdatedAt(),
	}
	self.data.Store(newData)
}

// ============================================================================
// LOCK-FREE GETTERS - All reads use atomic.Load(), no locks needed
// ============================================================================

// ID returns the session's ID.
func (self *ClientSession) ID() int64 {
	return self.data.Load().id
}

// UUID returns the session's UUID.
func (self *ClientSession) UUID() string {
	return self.data.Load().uuid
}

// DeviceID returns the device ID that owns this session.
func (self *ClientSession) DeviceID() int64 {
	return self.data.Load().devId
}

// Plugin returns the provider plugin of the session record.
func (self *ClientSession) Plugin() sdkapi.IPluginApi {
	providerPkg := self.data.Load().providerPkg
	if plugin, found := self.pluginsMgr.FindByPkg(providerPkg); found {
		return plugin
	}
	return nil
}

// Type returns the session type.
func (self *ClientSession) Type() sdkapi.SessionType {
	return sdkapi.SessionType(self.data.Load().sessionType)
}

// TimeSecs returns the session's available time in seconds.
func (self *ClientSession) TimeSecs() (sec int) {
	return self.data.Load().timeSecs
}

// DataMb returns the session's available data in megabytes.
func (self *ClientSession) DataMb() (mbytes float64) {
	return self.data.Load().dataMb
}

// TimeConsumption returns the session's time consumption in seconds.
// If session is currently running (resumed_at != nil), includes elapsed time since resumed_at.
func (self *ClientSession) TimeConsumption() (sec int) {
	d := self.data.Load()
	consumption := d.timeCons

	// If session is running, add elapsed time since resumed_at
	if d.resumedAt != nil {
		elapsed := int(time.Since(*d.resumedAt).Seconds())
		consumption += elapsed
	}

	return consumption
}

// DataConsumption returns the session's data consumption in megabytes.
func (self *ClientSession) DataConsumption() (mbytes float64) {
	return self.data.Load().dataCons
}

// ConsumedTimeSecs returns the raw stored time consumption in seconds.
// Does NOT include elapsed time since resumed_at.
func (self *ClientSession) ConsumedTimeSecs() (sec int) {
	return self.data.Load().timeCons
}

// ConsumedDataMb returns the raw stored data consumption in megabytes.
func (self *ClientSession) ConsumedDataMb() (mbytes float64) {
	return self.data.Load().dataCons
}

// RemainingTime returns the session's remaining time in seconds.
func (self *ClientSession) RemainingTime() (sec int) {
	d := self.data.Load()
	return self.remainingTimeWithData(d)
}

// RemainingData returns the session's remaining data in megabytes.
func (self *ClientSession) RemainingData() (mbytes float64) {
	d := self.data.Load()
	mb := self.remainingDataWithData(d)
	if mb < 0 {
		return 0
	}
	return mb
}

// ============================================================================
// INTERNAL HELPERS - Use provided data snapshot for consistent calculations
// ============================================================================

// isExpiredWithData checks expiration using provided data snapshot.
func (self *ClientSession) isExpiredWithData(d *sessionData) bool {
	if d.startedAt != nil && d.expDays != nil {
		exp := d.startedAt.Add(time.Hour * 24 * time.Duration(*d.expDays))
		return !time.Now().Before(exp)
	}
	return false
}

// remainingTimeWithData calculates remaining time using provided data snapshot.
func (self *ClientSession) remainingTimeWithData(d *sessionData) int {
	// If session has no consumption and was never started, return full allocated time
	hasConsumption := d.timeCons > 0 || d.dataCons > 0
	if d.startedAt == nil && d.resumedAt == nil && !hasConsumption {
		return d.timeSecs
	}

	remaining := d.timeSecs - d.timeCons

	// If session is running, subtract elapsed time since resumed_at
	if d.resumedAt != nil {
		elapsed := int(time.Since(*d.resumedAt).Seconds())
		remaining -= elapsed
	}

	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

// remainingDataWithData calculates remaining data using provided data snapshot.
func (self *ClientSession) remainingDataWithData(d *sessionData) float64 {
	// If session has no consumption and was never started, return full allocated data
	hasConsumption := d.timeCons > 0 || d.dataCons > 0
	if d.startedAt == nil && d.resumedAt == nil && !hasConsumption {
		return d.dataMb
	}

	remaining := d.dataMb - d.dataCons
	if remaining < 0 {
		return 0
	}
	return remaining
}

// expiresAtWithData calculates expiration time using provided data snapshot.
func (self *ClientSession) expiresAtWithData(d *sessionData) *time.Time {
	if d.startedAt != nil && d.expDays != nil {
		exp := d.startedAt.Add(time.Hour * 24 * time.Duration(*d.expDays))
		return &exp
	}
	return nil
}

// isAvailableWithData checks if session is available using provided data snapshot.
func (self *ClientSession) isAvailableWithData(d *sessionData) bool {
	if self.isExpiredWithData(d) {
		return false
	}
	hasConsumption := d.timeCons > 0 || d.dataCons > 0
	return d.startedAt == nil && d.resumedAt == nil && !hasConsumption
}

// isConsumedWithData checks if session is consumed using pre-computed values.
func (self *ClientSession) isConsumedWithData(d *sessionData, sessionType sdkapi.SessionType, remainingTime int, remainingData float64, isAvailable bool, isExpired bool) bool {
	// Available sessions are not consumed
	if isAvailable {
		return false
	}

	if isExpired {
		return true
	}

	if sessionType == sdkapi.SessionTypeTime || sessionType == sdkapi.SessionTypeTimeOrData {
		if remainingTime <= 0 {
			return true
		}
	}

	if sessionType == sdkapi.SessionTypeData || sessionType == sdkapi.SessionTypeTimeOrData {
		if remainingData <= 0 {
			return true
		}
	}

	return false
}

// IsConsumed returns true if the session resources are fully consumed.
// Returns false if the session has never been started (is available).
func (self *ClientSession) IsConsumed() bool {
	// Available sessions are not consumed
	if self.IsAvailable() {
		return false
	}

	d := self.data.Load()
	sessionType := sdkapi.SessionType(d.sessionType)

	if self.isExpiredWithData(d) {
		return true
	}

	if sessionType == sdkapi.SessionTypeTime || sessionType == sdkapi.SessionTypeTimeOrData {
		if self.remainingTimeWithData(d) <= 0 {
			return true
		}
	}

	if sessionType == sdkapi.SessionTypeData || sessionType == sdkapi.SessionTypeTimeOrData {
		if self.remainingDataWithData(d) <= 0 {
			return true
		}
	}

	return false
}

// IsExpired returns true if the session has passed its expiration date.
func (self *ClientSession) IsExpired() bool {
	return self.isExpiredWithData(self.data.Load())
}

// StartedAt returns the time when session was first started.
func (self *ClientSession) StartedAt() *time.Time {
	return self.data.Load().startedAt
}

// ResumedAt returns the time when session was last resumed.
func (self *ClientSession) ResumedAt() *time.Time {
	return self.data.Load().resumedAt
}

// IsRunning returns true if the session is currently active (resumedAt is not nil).
func (self *ClientSession) IsRunning() bool {
	return self.data.Load().resumedAt != nil
}

// IsAvailable returns true if the session is available for use.
// A session is NOT available if:
// - It has been started (started_at OR resumed_at is set, OR there's consumption data), OR
// - It has expired
func (self *ClientSession) IsAvailable() bool {
	if self.IsExpired() {
		return false
	}
	d := self.data.Load()
	hasConsumption := d.timeCons > 0 || d.dataCons > 0
	return d.startedAt == nil && d.resumedAt == nil && !hasConsumption
}

// CreatedAt returns the created at time.
func (self *ClientSession) CreatedAt() time.Time {
	return self.data.Load().createdAt
}

// UpdatedAt returns the updated at time.
func (self *ClientSession) UpdatedAt() time.Time {
	return self.data.Load().updatedAt
}

// ExpDays returns the session's expiration time in days.
func (self *ClientSession) ExpDays() *int {
	return self.data.Load().expDays
}

// ExpiresAt returns the time when session will expire.
func (self *ClientSession) ExpiresAt() *time.Time {
	d := self.data.Load()
	if d.startedAt != nil && d.expDays != nil {
		exp := d.startedAt.Add(time.Hour * 24 * time.Duration(*d.expDays))
		return &exp
	}
	return nil
}

// DownMbits returns the session's download speed limit in megabits per second.
func (self *ClientSession) DownMbits() int {
	return self.data.Load().downMbits
}

// UpMbits returns the session's upload speed limit in megabits per second.
func (self *ClientSession) UpMbits() int {
	return self.data.Load().upMbits
}

// UseGlobalSpeed returns whether session uses global speed limits.
func (self *ClientSession) UseGlobalSpeed() bool {
	return self.data.Load().useGlobal
}

// Data returns a snapshot of all session data fields with pre-computed values.
// TimeCons includes elapsed time for running sessions.
func (self *ClientSession) Data() sdkapi.SessionData {
	d := self.data.Load()

	// Calculate time consumption including elapsed time for running sessions
	timeCons := d.timeCons
	if d.resumedAt != nil {
		elapsed := int(time.Since(*d.resumedAt).Seconds())
		timeCons += elapsed
	}

	// Pre-compute all derived values
	sessionType := sdkapi.SessionType(d.sessionType)
	remainingTime := self.remainingTimeWithData(d)
	remainingData := self.remainingDataWithData(d)
	expiresAt := self.expiresAtWithData(d)
	isExpired := self.isExpiredWithData(d)
	isAvailable := self.isAvailableWithData(d)
	isRunning := d.resumedAt != nil
	isConsumed := self.isConsumedWithData(d, sessionType, remainingTime, remainingData, isAvailable, isExpired)

	return sdkapi.SessionData{
		ID:             d.id,
		UUID:           d.uuid,
		DeviceID:       d.devId,
		Type:           sessionType,
		TimeSecs:       d.timeSecs,
		DataMb:         d.dataMb,
		TimeCons:       timeCons,
		DataCons:       d.dataCons,
		DownMbits:      d.downMbits,
		UpMbits:        d.upMbits,
		UseGlobalSpeed: d.useGlobal,
		ExpDays:        copyIntPtr(d.expDays),
		StartedAt:      copyTimePtr(d.startedAt),
		ResumedAt:      copyTimePtr(d.resumedAt),
		CreatedAt:      d.createdAt,
		UpdatedAt:      d.updatedAt,
		// Pre-computed values
		RemainingTime: remainingTime,
		RemainingData: remainingData,
		ExpiresAt:     expiresAt,
		IsExpired:     isExpired,
		IsAvailable:   isAvailable,
		IsConsumed:    isConsumed,
		IsRunning:     isRunning,
	}
}

// ============================================================================
// SETTERS - Use copy-modify-swap pattern with mutex protection
// ============================================================================

// IncTimeCons increases the session's time consumption in seconds.
// This value is not saved until Save() method is called.
func (self *ClientSession) IncTimeCons(sec int) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	newData.timeCons += sec
	newData.dirtyTime = true
	self.data.Store(&newData)
}

// IncDataCons increases the session's data consumption in megabytes.
// This value is not saved until Save() method is called.
func (self *ClientSession) IncDataCons(mbytes float64) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	newData.dataCons += mbytes
	newData.dirtyData = true
	self.data.Store(&newData)
}

// SetTimeSecs sets the session's available time in seconds.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetTimeSecs(sec int) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	newData.timeSecs = sec
	newData.dirtyTime = true
	self.data.Store(&newData)
}

// SetDataMb sets the session's available data in megabytes.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetDataMb(mbytes float64) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	newData.dataMb = mbytes
	newData.dirtyData = true
	self.data.Store(&newData)
}

// SetTimeCons sets the session's time consumption in seconds.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetTimeCons(sec int) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	newData.timeCons = sec
	newData.dirtyTime = true
	self.data.Store(&newData)
}

// SetDataCons sets the session's data consumption in megabytes.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetDataCons(mbytes float64) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	newData.dataCons = mbytes
	newData.dirtyData = true
	self.data.Store(&newData)
}

// SetStartedAt sets the time when session was first started.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetStartedAt(started *time.Time) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	newData.startedAt = copyTimePtr(started)
	self.data.Store(&newData)
}

// SetResumedAt sets the time when session was last resumed.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetResumedAt(resumed *time.Time) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	newData.resumedAt = copyTimePtr(resumed)
	self.data.Store(&newData)
}

// SetExpDays sets the session's expiration time in days.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetExpDays(exp *int) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	newData.expDays = copyIntPtr(exp)
	self.data.Store(&newData)
}

// SetDownMbits sets the session's download speed limit in megabits per second.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetDownMbits(mbits int) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	newData.downMbits = mbits
	newData.dirtyBandwidth = true
	self.data.Store(&newData)
}

// SetUpMbits sets the session's upload speed limit in megabits per second.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetUpMbits(mbits int) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	newData.upMbits = mbits
	newData.dirtyBandwidth = true
	self.data.Store(&newData)
}

// SetUseGlobalSpeed sets whether session uses global speed limits.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetUseGlobalSpeed(useGlobal bool) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	newData.useGlobal = useGlobal
	newData.dirtyBandwidth = true
	self.data.Store(&newData)
}

// ============================================================================
// INTERNAL PERSISTENCE METHODS - For bookkeeping operations
// ============================================================================

// PersistToDB saves the current session state directly to the database.
// Unlike Save(), this does NOT trigger the onSave callback and does NOT clear dirty flags.
// Used for internal bookkeeping operations (periodic saves, stop operations).
func (self *ClientSession) PersistToDB(ctx context.Context) error {
	d := self.data.Load()
	return self.mdls.Session().Update(ctx, models.UpdateSessionParams{
		ID:          d.id,
		UUID:        d.uuid,
		ProviderPkg: d.providerPkg,
		DeviceID:    d.devId,
		SessionType: sdkapi.SessionType(d.sessionType),
		TimeSecs:    d.timeSecs,
		DataMbytes:  d.dataMb,
		TimeCons:    d.timeCons,
		DataCons:    d.dataCons,
		StartedAt:   d.startedAt,
		ResumedAt:   d.resumedAt,
		ExpDays:     d.expDays,
		DownMbits:   d.downMbits,
		UpMbits:     d.upMbits,
		UseGlobal:   d.useGlobal,
	})
}

// SnapshotTimeCons atomically bakes elapsed time into timeCons and resets resumedAt.
// If clearResumed is true, sets resumedAt to nil (session stopping).
// If clearResumed is false, resets resumedAt to now (checkpoint for continued tracking).
// Returns elapsed seconds for logging purposes.
// Does NOT set dirty flags (internal bookkeeping operation).
func (self *ClientSession) SnapshotTimeCons(clearResumed bool) int {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	d := self.data.Load()
	if d.resumedAt == nil {
		return 0
	}

	elapsed := int(time.Since(*d.resumedAt).Seconds())
	newData := d.copy()
	newData.timeCons = d.timeCons + elapsed

	if clearResumed {
		newData.resumedAt = nil
	} else {
		now := time.Now().UTC()
		newData.resumedAt = &now
	}

	self.data.Store(&newData)
	return elapsed
}
