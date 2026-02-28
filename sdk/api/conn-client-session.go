/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"context"
	"time"
)

// SessionChangedFields tracks which session fields were modified since last save.
type SessionChangedFields struct {
	Time      bool // timeSecs or timeCons changed
	Data      bool // dataMb or dataCons changed
	Bandwidth bool // downMbits, upMbits, or useGlobal changed
}

// SessionSaveParams contains parameters for the session save callback.
type SessionSaveParams struct {
	Ctx           context.Context
	Session       IClientSession
	ChangedFields SessionChangedFields
}

// SessionSaveCallback is called after a session is saved to apply side effects.
// This allows the SessionsMgr to update running sessions (reset timers, update TC rules)
// and emit events when session.Save() is called.
type SessionSaveCallback func(params SessionSaveParams) error

// SessionData holds all session fields returned by Data() method.
// This struct is returned as a snapshot to minimize mutex usage.
type SessionData struct {
	ID             int64
	UUID           string
	DeviceID       int64
	Type           SessionType
	TimeSecs       int
	DataMb         float64
	TimeCons       int     // Raw stored time consumption (without elapsed)
	DataCons       float64 // Raw stored data consumption
	DownMbits      int
	UpMbits        int
	UseGlobalSpeed bool
	ExpDays        *int
	StartedAt      *time.Time
	ResumedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RemainingTime returns the session's remaining time in seconds.
func (s SessionData) RemainingTime() int {
	// If session has no consumption and was never started, return full allocated time
	hasConsumption := s.TimeCons > 0 || s.DataCons > 0
	if s.StartedAt == nil && s.ResumedAt == nil && !hasConsumption {
		return s.TimeSecs
	}

	remaining := s.TimeSecs - s.TimeCons
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RemainingData returns the session's remaining data in megabytes.
func (s SessionData) RemainingData() float64 {
	// If session has no consumption and was never started, return full allocated data
	hasConsumption := s.TimeCons > 0 || s.DataCons > 0
	if s.StartedAt == nil && s.ResumedAt == nil && !hasConsumption {
		return s.DataMb
	}

	remaining := s.DataMb - s.DataCons
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ExpiresAt returns the time when session will expire, or nil if no expiration.
func (s SessionData) ExpiresAt() *time.Time {
	if s.ExpDays == nil {
		return nil
	}
	// Use started_at, or resumed_at as fallback if started_at is nil
	effectiveStart := s.StartedAt
	if effectiveStart == nil {
		effectiveStart = s.ResumedAt
	}
	if effectiveStart == nil {
		return nil
	}
	expTime := effectiveStart.Add(time.Hour * 24 * time.Duration(*s.ExpDays))
	return &expTime
}

// IsExpired returns true if the session has passed its expiration date.
func (s SessionData) IsExpired() bool {
	expiresAt := s.ExpiresAt()
	if expiresAt == nil {
		return false
	}
	return time.Now().After(*expiresAt)
}

// IsAvailable returns true if the session is available for use.
// A session is NOT available if:
// - It has been started (started_at OR resumed_at is set, OR there's consumption data), OR
// - It has expired
func (s SessionData) IsAvailable() bool {
	if s.IsExpired() {
		return false
	}
	hasConsumption := s.TimeCons > 0 || s.DataCons > 0
	return s.StartedAt == nil && s.ResumedAt == nil && !hasConsumption
}

// IsConsumed returns true if the session resources are fully consumed or expired.
// Returns false if the session has never been started (is available).
func (s SessionData) IsConsumed() bool {
	// Available sessions are not consumed
	if s.IsAvailable() {
		return false
	}

	if s.IsExpired() {
		return true
	}

	switch s.Type {
	case SessionTypeTime:
		return s.RemainingTime() <= 0
	case SessionTypeData:
		return s.RemainingData() <= 0
	case SessionTypeTimeOrData:
		return s.RemainingTime() <= 0 || s.RemainingData() <= 0
	default:
		return s.RemainingTime() <= 0
	}
}

// IsRunning returns true if the session is currently active (resumedAt is not nil).
func (s SessionData) IsRunning() bool {
	return s.ResumedAt != nil
}

// IClientSession represents a client's internet connection session.
type IClientSession interface {
	// Returns the session's ID.
	ID() int64

	// Returns the session's UUID.
	UUID() string

	// Returns the device ID that owns this session.
	DeviceID() int64

	// Returns the provider plugin of the session record.
	Plugin() IPluginApi

	// Returns the session type.
	Type() SessionType

	// Return the session's available time in seconds.
	TimeSecs() (sec int)

	// Returns the session's available data in megabytes.
	DataMb() (mbytes float64)

	// Returns the session's time consumption in seconds.
	// If session is running, includes elapsed time since resumed_at.
	TimeConsumption() (sec int)

	// Returns the session's data consumption in megabytes.
	DataConsumption() (mbytes float64)

	// Returns the raw stored time consumption in seconds (without elapsed calculation).
	// Use this for syncing/persistence where you need the base value.
	ConsumedTimeSecs() (sec int)

	// Returns the raw stored data consumption in megabytes.
	// Use this for syncing/persistence where you need the base value.
	ConsumedDataMb() (mbytes float64)

	// Returns the session's remaining time in seconds.
	RemainingTime() (sec int)

	// Returns the session's remaining data in megabytes.
	RemainingData() (mbytes float64)

	// Returns true if the session resources are fully consumed.
	IsConsumed() bool

	// Returns true if the session has passed its expiration date.
	IsExpired() bool

	// Returns the time when session was first started.
	StartedAt() *time.Time

	// Returns the time when session was last resumed.
	ResumedAt() *time.Time

	// Returns the created at time.
	CreatedAt() time.Time

	// Returns the updated at time.
	UpdatedAt() time.Time

	// Returns the session's expiration time in days.
	// If session has no expiration, it returns nil.
	ExpDays() *int

	// Returns the time when session will expire.
	// If session has no expiration, it returns nil.
	ExpiresAt() *time.Time

	// Returns the session's download speed limit in megabits per second.
	DownMbits() int

	// Returns the session's upload speed limit in megabits per second.
	UpMbits() int

	// Returns whether session uses global speed limits.
	UseGlobalSpeed() bool

	// IsRunning returns true if the session is currently active (resumedAt is not nil).
	IsRunning() bool

	// IsAvailable returns true if the session has never been started (available for use).
	IsAvailable() bool

	// Returns a snapshot of all session data fields.
	// This method acquires the mutex once and returns all fields,
	// reducing lock contention compared to calling individual getters.
	// Note: TimeCons includes elapsed time for running sessions.
	Data() SessionData

	// Returns a snapshot of all session data fields with raw stored values.
	// Unlike Data(), TimeCons does NOT include elapsed time calculation.
	// Use this for syncing/persistence where you need the base values.
	RawData() SessionData

	// Increases the session's time consumption in seconds.
	// This value is not saved until Save() method is called.
	IncTimeCons(sec int)

	// Increases the session's data consumption in megabytes.
	// This value is not saved until Save() method is called.
	IncDataCons(mbytes float64)

	// Sets the session's available time in seconds.
	// This value is not saved until Save() method is called.
	SetTimeSecs(sec int)

	// Sets the session's available data in megabytes.
	// This value is not saved until Save() method is called.
	SetDataMb(mbytes float64)

	// Sets the session's time consumption in seconds.
	// This value is not saved until Save() method is called.
	SetTimeCons(sec int)

	// Sets the session's data consumption in megabytes.
	// This value is not saved until Save() method is called.
	SetDataCons(mbytes float64)

	// Sets the time when session was first started.
	// This value is not saved until Save() method is called.
	SetStartedAt(started *time.Time)

	// Sets the time when session was last resumed.
	// This value is not saved until Save() method is called.
	SetResumedAt(resumed *time.Time)

	// Sets the session's expiration time in days.
	// This value is not saved until Save() method is called.
	SetExpDays(exp *int)

	// Sets the session's download speed limit in megabits per second.
	// This value is not saved until Save() method is called.
	SetDownMbits(mbits int)

	// Sets the session's upload speed limit in megabits per second.
	// This value is not saved until Save() method is called.
	SetUpMbits(mbits int)

	// Sets whether session uses global speed limits.
	// This value is not saved until Save() method is called.
	SetUseGlobalSpeed(bool)

	// Saves the session's changes.
	Save(ctx context.Context) error

	// Reloads the session's data from the database.
	Reload(ctx context.Context) error

	// Saves the session state directly to the database without triggering save callbacks.
	// Unlike Save(), this does NOT trigger the onSave callback and does NOT clear dirty flags.
	// Used for internal bookkeeping operations (periodic saves, stop operations).
	PersistToDB(ctx context.Context) error

	// Atomically snapshots elapsed time into stored consumption and resets resumedAt.
	// If clearResumed is true, sets resumedAt to nil (session stopping).
	// If clearResumed is false, resets resumedAt to now (checkpoint for continued tracking).
	// Returns elapsed seconds for logging purposes.
	// Does NOT set dirty flags (internal bookkeeping operation).
	SnapshotTimeCons(clearResumed bool) int
}
