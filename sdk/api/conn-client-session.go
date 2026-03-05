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
// Maps directly to database columns for granular change tracking.
type SessionChangedFields struct {
	TimeSecs       bool // time_secs: Allocated time in seconds
	DataMb         bool // data_mb: Allocated data in megabytes
	TimeCons       bool // time_secs_consumed: Time consumption in seconds
	DataCons       bool // data_mb_consumed: Data consumption in megabytes
	DownMbits      bool // down_speed_mbits: Download speed limit in Mbps
	UpMbits        bool // up_speed_mbits: Upload speed limit in Mbps
	UseGlobalSpeed bool // use_global_speed: Whether to use global speed settings
	ExpDays        bool // exp_days: Expiration days (nullable)
	StartedAt      bool // started_at: When session was first started (nullable)
	ResumedAt      bool // resumed_at: When session was last resumed (nullable)
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

// SessionRawData holds raw session fields as stored in the database.
// Use this for syncing/persistence where you need exact stored values.
type SessionRawData struct {
	ID             int64
	UUID           string
	DeviceID       int64
	Type           SessionType
	TimeSecs       int        // Allocated time in seconds
	DataMb         float64    // Allocated data in megabytes
	TimeCons       int        // Time consumption in seconds (raw stored value)
	DataCons       float64    // Data consumption in megabytes (raw stored value)
	DownMbits      int        // Download speed limit in Mbps
	UpMbits        int        // Upload speed limit in Mbps
	UseGlobalSpeed bool       // Whether to use global speed settings
	ExpDays        *int       // Expiration days (nil if no expiration)
	StartedAt      *time.Time // When session was first started
	ResumedAt      *time.Time // When session was last resumed (nil if not running)
	CreatedAt      time.Time  // Creation timestamp
	UpdatedAt      time.Time  // Last update timestamp
}

// SessionData holds all session fields with pre-computed values.
// TimeCons includes elapsed time for running sessions.
// Use this for display/logic where you need current state.
type SessionData struct {
	// Raw database values
	ID             int64
	UUID           string
	DeviceID       int64
	Type           SessionType
	TimeSecs       int        // Allocated time in seconds
	DataMb         float64    // Allocated data in megabytes
	TimeCons       int        // Time consumption in seconds (includes elapsed time if running)
	DataCons       float64    // Data consumption in megabytes
	DownMbits      int        // Download speed limit in Mbps
	UpMbits        int        // Upload speed limit in Mbps
	UseGlobalSpeed bool       // Whether to use global speed settings
	ExpDays        *int       // Expiration days (nil if no expiration)
	StartedAt      *time.Time // When session was first started
	ResumedAt      *time.Time // When session was last resumed (nil if not running)
	CreatedAt      time.Time  // Creation timestamp
	UpdatedAt      time.Time  // Last update timestamp

	// Pre-computed values
	RemainingTime int        // Remaining time in seconds
	RemainingData float64    // Remaining data in megabytes
	ExpiresAt     *time.Time // When session expires, or nil if no expiration
	IsExpired     bool       // True if session has passed expiration date
	IsAvailable   bool       // True if session has never been started
	IsConsumed    bool       // True if session resources are fully consumed
	IsRunning     bool       // True if session is currently active
}

// SessionUpdateData contains fields to update on a session in a single batch operation.
// Only non-nil pointer fields will be updated. This allows selective field updates.
// Use SetData() to apply all updates in a single lock acquisition.
type SessionUpdateData struct {
	TimeSecs       *int       // Allocated time in seconds
	DataMb         *float64   // Allocated data in megabytes
	TimeCons       *int       // Time consumption in seconds
	DataCons       *float64   // Data consumption in megabytes
	DownMbits      *int       // Download speed limit in Mbps
	UpMbits        *int       // Upload speed limit in Mbps
	UseGlobalSpeed *bool      // Whether to use global speed settings
	StartedAt      *time.Time // When session was first started (nil clears it)
	ResumedAt      *time.Time // When session was last resumed (nil clears it)
	ExpDays        *int       // Expiration days (nil clears it)
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

	// Returns a snapshot of all session data fields with pre-computed values.
	// This method acquires the mutex once and returns all fields,
	// reducing lock contention compared to calling individual getters.
	// TimeCons includes elapsed time for running sessions.
	// Pre-computed fields: RemainingTime, RemainingData, ExpiresAt, IsExpired, IsAvailable, IsConsumed, IsRunning.
	Data() SessionData

	// Returns a snapshot of raw session data fields as stored in the database.
	// Use this for syncing/persistence where you need exact stored values.
	// TimeCons does NOT include elapsed time - it's the raw stored value.
	RawData() SessionRawData

	// Increases the session's time consumption in seconds.
	// This value is not saved until Save() method is called.
	IncTimeCons(sec int)

	// Increases the session's data consumption in megabytes.
	// This value is not saved until Save() method is called.
	IncDataCons(mbytes float64)

	// Sets multiple session fields in a single batch operation.
	// Only non-nil fields in the data parameter will be updated.
	// Values are not saved until Save() method is called.
	SetData(data SessionUpdateData)

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

	// Sync reloads session data from the database and applies any changes to the running session.
	// This is useful when session data has been modified externally (e.g., by another process or
	// direct database update) and you need to synchronize the in-memory state.
	// For running sessions, this will:
	// - Reset the timer if time allocation changed
	// - Update TC (traffic control) rules if bandwidth settings changed
	// - Stop the session if resources are now consumed
	// Emits EventSessionChanged after syncing.
	Sync(ctx context.Context) error
}
