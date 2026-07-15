package sessmgr

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"core/db"
	"core/db/models"
	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// =============================================================================
// TYPES
// =============================================================================

// ClientSession wraps session data with lock-free reads and synchronized writes.
// Reads use atomic.Pointer for zero-lock access.
// Writes use copy-modify-swap pattern protected by writeMu.
type ClientSession struct {
	// Dependencies - immutable after creation, no lock needed
	db         *db.Database
	mdls       *models.Models
	pluginsMgr sdkapi.IPluginsMgrApi
	sessMgr    *SessionsMgr

	// Session data - atomic pointer for lock-free reads
	data atomic.Pointer[sessionData]

	// Write mutex - only needed for modifications (copy-modify-swap)
	writeMu sync.Mutex
}

// =============================================================================
// CONSTRUCTOR
// =============================================================================

func NewClientSession(dtb *db.Database, mdls *models.Models, pluginsMgr sdkapi.IPluginsMgrApi, sessMgr *SessionsMgr, s *models.Session) *ClientSession {
	cs := &ClientSession{
		db:         dtb,
		mdls:       mdls,
		pluginsMgr: pluginsMgr,
		sessMgr:    sessMgr,
	}
	cs.loadFromModel(s)
	return cs
}

// =============================================================================
// PUBLIC METHODS - Persistence
// =============================================================================

func (self *ClientSession) Save(ctx context.Context, opts *sdkapi.SessionSaveOpts) error {
	// Take a consistent snapshot and clear dirty flags atomically.
	// This ensures no writes are lost between reading flags and clearing them.
	self.writeMu.Lock()
	d := self.data.Load()

	// Collect changed fields from the current snapshot
	changedFields := d.changedFields()

	// Clear dirty flags immediately so concurrent setters mark new changes
	hasChanges := hasAnyChange(changedFields)

	if hasChanges {
		newData := d.copy()
		newData.clearDirtyFlags()
		self.data.Store(&newData)
	}
	self.writeMu.Unlock()

	// Save to database (outside lock - DB operations can be slow)
	err := self.mdls.Session().Update(ctx, d.updateParams())
	if err != nil {
		return err
	}

	// Apply side effects if any fields changed
	if hasChanges {
		return self.handleChanges(ctx, changedFields, opts)
	}

	return nil
}

// UpdateAndSave atomically applies update data and persists it to the database
// in a single writeMu window (a fused SetData+Save). The new snapshot is only
// stored in memory after the DB write succeeds, so on error nothing is applied
// and pre-existing dirty fields (e.g. unsaved IncDataCons increments) survive
// untouched. Pre-existing dirty fields are persisted along with the update,
// their flags cleared, and reported in the changed set — mirroring Save().
//
// handleChanges runs outside writeMu: handleSessionSaved's ApplyTimeUpdate /
// ApplyBandwidthUpdate re-acquire writeMu (via SnapshotTimeCons / SetData) and
// would self-deadlock if invoked while the lock is still held.
func (self *ClientSession) UpdateAndSave(ctx context.Context, data sdkapi.SessionUpdateData, opts *sdkapi.SessionSaveOpts) error {
	self.writeMu.Lock()

	old := self.data.Load()
	newData := old.copy()
	applyUpdateData(&newData, data)

	// Changed set = fields set by this update + previously unsaved changes.
	changedFields := newData.changedFields()
	hasChanges := hasAnyChange(changedFields)
	newData.clearDirtyFlags()

	// Persist while still holding writeMu so no concurrent writer can
	// interleave between the DB write and the in-memory swap. writeMu is a
	// leaf lock (the models layer takes no sessmgr locks), so holding it
	// across the write adds only bounded per-session contention.
	if err := self.mdls.Session().Update(ctx, newData.updateParams()); err != nil {
		self.writeMu.Unlock()
		return err
	}

	self.data.Store(&newData)
	self.writeMu.Unlock()

	if hasChanges {
		return self.handleChanges(ctx, changedFields, opts)
	}

	return nil
}

// PersistToDB saves the current session state directly to the database.
// Unlike Save(), this does NOT trigger the onSave callback and does NOT clear dirty flags.
// Used for internal bookkeeping operations (periodic saves, stop operations).
func (self *ClientSession) PersistToDB(ctx context.Context) error {
	return self.mdls.Session().Update(ctx, self.data.Load().updateParams())
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

	// If counter is paused, elapsed time was already snapshotted by Pause()
	elapsed := 0
	if d.pausedAt == nil {
		elapsed = int(time.Since(*d.resumedAt).Round(time.Second).Seconds())
	}
	newData := d.copy()
	newData.timeCons = d.timeCons + elapsed

	if clearResumed {
		newData.resumedAt = nil
		newData.pausedAt = nil
	} else if d.pausedAt == nil {
		now := time.Now().UTC()
		newData.resumedAt = &now
	}

	self.data.Store(&newData)
	return elapsed
}

// Pause stops both time and data counters by snapshotting elapsed time into
// stored consumption and setting pausedAt. The session remains connected
// (resumedAt is preserved, TC rules and bandwidth limits stay active) but no
// further time or data is counted. Caller must call PersistToDB() to persist the snapshot.
func (self *ClientSession) Pause() {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	d := self.data.Load()
	if d.resumedAt == nil || d.pausedAt != nil {
		return
	}

	elapsed := int(time.Since(*d.resumedAt).Round(time.Second).Seconds())
	now := time.Now().UTC()
	newData := d.copy()
	newData.timeCons = d.timeCons + elapsed
	newData.pausedAt = &now
	self.data.Store(&newData)
}

// Resume resumes both time and data counters after they were paused by Pause().
// Clears pausedAt and resets resumedAt to now so elapsed time calculation starts
// fresh from this point. Data consumption is counted again from here forward.
func (self *ClientSession) Resume() {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	d := self.data.Load()
	if d.pausedAt == nil {
		return
	}

	now := time.Now().UTC()
	newData := d.copy()
	newData.pausedAt = nil
	newData.resumedAt = &now
	self.data.Store(&newData)
}

// IsPaused returns true if the time/data counters are paused (Pause() was called
// and Resume() has not been called since). While paused the session stays
// connected but no time or data is counted.
func (self *ClientSession) IsPaused() bool {
	return self.data.Load().pausedAt != nil
}

// Status returns the session's current status: running, paused, or stopped.
func (self *ClientSession) Status() sdkapi.ClientSessionStatus {
	d := self.data.Load()
	if d.resumedAt == nil {
		return sdkapi.ClielntSessionStatusStopped
	}
	if d.pausedAt != nil {
		return sdkapi.ClientSessionStatusPaused
	}
	return sdkapi.ClientSessionStatusRunning
}

// =============================================================================
// PUBLIC METHODS - Getters (Lock-free reads via atomic.Load())
// =============================================================================

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
// If session is currently running (resumed_at != nil) and counter is not paused,
// includes elapsed time since resumed_at.
func (self *ClientSession) TimeConsumption() (sec int) {
	d := self.data.Load()
	consumption := d.timeCons

	// If session is running and counter is not paused, add elapsed time since resumed_at
	if d.resumedAt != nil && d.pausedAt == nil {
		elapsed := int(time.Since(*d.resumedAt).Round(time.Second).Seconds())
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
// TimeCons includes elapsed time for running sessions (unless counter is paused).
func (self *ClientSession) Data() sdkapi.SessionData {
	d := self.data.Load()

	// Calculate time consumption including elapsed time for running sessions
	timeCons := d.timeCons
	if d.resumedAt != nil && d.pausedAt == nil {
		elapsed := int(time.Since(*d.resumedAt).Round(time.Second).Seconds())
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
	isPaused := d.pausedAt != nil
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
		ExpDays:        sdkutils.CopyIntPtr(d.expDays),
		StartedAt:      sdkutils.CopyTimePtr(d.startedAt),
		ResumedAt:      sdkutils.CopyTimePtr(d.resumedAt),
		PausedAt:       sdkutils.CopyTimePtr(d.pausedAt),
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
		IsPaused:      isPaused,
	}
}

// RawData returns a snapshot of raw session data fields as stored in the database.
// TimeCons does NOT include elapsed time - it's the raw stored value.
// Use this for syncing/persistence where you need exact stored values.
func (self *ClientSession) RawData() sdkapi.SessionRawData {
	d := self.data.Load()

	return sdkapi.SessionRawData{
		ID:             d.id,
		UUID:           d.uuid,
		DeviceID:       d.devId,
		Type:           sdkapi.SessionType(d.sessionType),
		TimeSecs:       d.timeSecs,
		DataMb:         d.dataMb,
		TimeCons:       d.timeCons,
		DataCons:       d.dataCons,
		DownMbits:      d.downMbits,
		UpMbits:        d.upMbits,
		UseGlobalSpeed: d.useGlobal,
		ExpDays:        sdkutils.CopyIntPtr(d.expDays),
		StartedAt:      sdkutils.CopyTimePtr(d.startedAt),
		ResumedAt:      sdkutils.CopyTimePtr(d.resumedAt),
		PausedAt:       sdkutils.CopyTimePtr(d.pausedAt),
		CreatedAt:      d.createdAt,
		UpdatedAt:      d.updatedAt,
	}
}

// =============================================================================
// PUBLIC METHODS - Setters (copy-modify-swap pattern with mutex protection)
// =============================================================================

// IncTimeCons increases the session's time consumption in seconds.
// This value is not saved until Save() method is called.
func (self *ClientSession) IncTimeCons(sec int) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	newData.timeCons += sec
	newData.dirtyTimeCons = true
	self.data.Store(&newData)
}

// IncDataCons increases the session's data consumption in megabytes.
// This value is not saved until Save() method is called.
// No-op when the counter is paused (Pause was called).
func (self *ClientSession) IncDataCons(mbytes float64) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	if old.pausedAt != nil {
		return
	}
	newData := old.copy()
	newData.dataCons += mbytes
	newData.dirtyDataCons = true
	self.data.Store(&newData)
}

// SetData sets multiple session fields in a single batch operation.
// Only non-nil fields in the data parameter will be updated.
// This is more efficient than calling individual setters when updating multiple fields.
// This value is not saved until Save() method is called.
func (self *ClientSession) SetData(data sdkapi.SessionUpdateData) {
	self.writeMu.Lock()
	defer self.writeMu.Unlock()

	old := self.data.Load()
	newData := old.copy()
	applyUpdateData(&newData, data)
	self.data.Store(&newData)
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

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
		startedAt:   sdkutils.CopyTimePtr(d.startedAt),
		resumedAt:   sdkutils.CopyTimePtr(d.resumedAt),
		pausedAt:    sdkutils.CopyTimePtr(d.pausedAt),
		expDays:     sdkutils.CopyIntPtr(d.expDays),
		downMbits:   d.downMbits,
		upMbits:     d.upMbits,
		useGlobal:   d.useGlobal,
		createdAt:   d.createdAt,
		updatedAt:   d.updatedAt,
		// Preserve dirty flags during copy
		dirtyTimeSecs:       d.dirtyTimeSecs,
		dirtyDataMb:         d.dirtyDataMb,
		dirtyTimeCons:       d.dirtyTimeCons,
		dirtyDataCons:       d.dirtyDataCons,
		dirtyDownMbits:      d.dirtyDownMbits,
		dirtyUpMbits:        d.dirtyUpMbits,
		dirtyUseGlobalSpeed: d.dirtyUseGlobalSpeed,
		dirtyExpDays:        d.dirtyExpDays,
		dirtyStartedAt:      d.dirtyStartedAt,
		dirtyResumedAt:      d.dirtyResumedAt,
	}
}

// applyUpdateData applies non-nil fields from data onto d, marking each
// applied field's dirty flag. Shared by SetData and UpdateAndSave so the
// field mapping cannot drift between the two paths.
func applyUpdateData(d *sessionData, data sdkapi.SessionUpdateData) {
	// Update time/data allocation and consumption
	if data.TimeSecs != nil {
		d.timeSecs = *data.TimeSecs
		d.dirtyTimeSecs = true
	}
	if data.DataMb != nil {
		d.dataMb = *data.DataMb
		d.dirtyDataMb = true
	}
	if data.TimeCons != nil {
		d.timeCons = *data.TimeCons
		d.dirtyTimeCons = true
	}
	if data.DataCons != nil {
		d.dataCons = *data.DataCons
		d.dirtyDataCons = true
	}

	// Update bandwidth settings
	if data.DownMbits != nil {
		d.downMbits = *data.DownMbits
		d.dirtyDownMbits = true
	}
	if data.UpMbits != nil {
		d.upMbits = *data.UpMbits
		d.dirtyUpMbits = true
	}
	if data.UseGlobalSpeed != nil {
		d.useGlobal = *data.UseGlobalSpeed
		d.dirtyUseGlobalSpeed = true
	}

	// Update timestamps
	if data.StartedAt != nil {
		d.startedAt = sdkutils.CopyTimePtr(data.StartedAt)
		d.dirtyStartedAt = true
	}
	if data.ResumedAt != nil {
		d.resumedAt = sdkutils.CopyTimePtr(data.ResumedAt)
		d.dirtyResumedAt = true
	}
	if data.ExpDays != nil {
		d.expDays = sdkutils.CopyIntPtr(data.ExpDays)
		d.dirtyExpDays = true
	}
}

// changedFields collects the SessionChangedFields recorded by d's dirty flags.
func (d *sessionData) changedFields() sdkapi.SessionChangedFields {
	return sdkapi.SessionChangedFields{
		TimeSecs:       d.dirtyTimeSecs,
		DataMb:         d.dirtyDataMb,
		TimeCons:       d.dirtyTimeCons,
		DataCons:       d.dirtyDataCons,
		DownMbits:      d.dirtyDownMbits,
		UpMbits:        d.dirtyUpMbits,
		UseGlobalSpeed: d.dirtyUseGlobalSpeed,
		ExpDays:        d.dirtyExpDays,
		StartedAt:      d.dirtyStartedAt,
		ResumedAt:      d.dirtyResumedAt,
	}
}

// clearDirtyFlags resets all dirty flags on d.
func (d *sessionData) clearDirtyFlags() {
	d.dirtyTimeSecs = false
	d.dirtyDataMb = false
	d.dirtyTimeCons = false
	d.dirtyDataCons = false
	d.dirtyDownMbits = false
	d.dirtyUpMbits = false
	d.dirtyUseGlobalSpeed = false
	d.dirtyExpDays = false
	d.dirtyStartedAt = false
	d.dirtyResumedAt = false
}

// updateParams builds the model update params from a sessionData snapshot.
// Shared by Save, PersistToDB, and UpdateAndSave so the field lists cannot drift.
func (d *sessionData) updateParams() models.UpdateSessionParams {
	return models.UpdateSessionParams{
		ID:             d.id,
		UUID:           d.uuid,
		ProviderPkg:    d.providerPkg,
		Type:           sdkapi.SessionType(d.sessionType),
		TimeSecs:       d.timeSecs,
		DataMb:         d.dataMb,
		TimeCons:       d.timeCons,
		DataCons:       d.dataCons,
		StartedAt:      d.startedAt,
		ResumedAt:      d.resumedAt,
		PausedAt:       d.pausedAt,
		ExpDays:        d.expDays,
		DownMbits:      d.downMbits,
		UpMbits:        d.upMbits,
		UseGlobalSpeed: d.useGlobal,
	}
}

// hasAnyChange reports whether any field in the changed set is marked.
func hasAnyChange(f sdkapi.SessionChangedFields) bool {
	return f.TimeSecs || f.DataMb || f.TimeCons || f.DataCons ||
		f.DownMbits || f.UpMbits || f.UseGlobalSpeed || f.ExpDays ||
		f.StartedAt || f.ResumedAt
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
		expDays:     sdkutils.CopyIntPtr(s.ExpDays()),
		startedAt:   sdkutils.CopyTimePtr(s.StartedAt()),
		resumedAt:   sdkutils.CopyTimePtr(s.ResumedAt()),
		pausedAt:    sdkutils.CopyTimePtr(s.PausedAt()),
		createdAt:   s.CreatedAt(),
		updatedAt:   s.UpdatedAt(),
	}
	self.data.Store(newData)
}

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

	// If session is running and counter is not paused, subtract elapsed time since resumed_at
	if d.resumedAt != nil && d.pausedAt == nil {
		elapsed := int(time.Since(*d.resumedAt).Round(time.Second).Seconds())
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

// handleChanges is called after Save()/UpdateAndSave() to apply side effects to running sessions.
func (self *ClientSession) handleChanges(ctx context.Context, changed sdkapi.SessionChangedFields, opts *sdkapi.SessionSaveOpts) error {
	return self.sessMgr.handleSessionSaved(ctx, self, changed, opts)
}
