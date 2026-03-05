# IClientSession

The `IClientSession` represents a session for the [IClientDevice](./client-device.md). It can be manipulated using the [SessionsMgrApi](./sessions-mgr-api.md). To get an instance of `IClientSession`, you can use the [CreateSession](./sessions-mgr-api.md#createsession), [RunningSession](./sessions-mgr-api.md#runningsession) or [AvailableSession](./sessions-mgr-api.md#availablesession) methods from [SessionMgrApi](./sessions-mgr-api.md). For example:

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)
    session, _ := api.SessionsMgr().RunningSession(clnt)
}
```

## Supporting Types

### SessionType

`SessionType` is a string type that defines the type of session. The available values are:

| Value | Description |
| --- | --- |
| `"time"` | A time-based session that expires when the allocated time is consumed |
| `"data"` | A data-based session that expires when the allocated data is consumed |
| `"time-or-data"` | A session limited by both time and data, expiring when either is exhausted |

### SessionChangedFields

Tracks which session fields were modified since the last save. Maps directly to database columns for granular change tracking. Used internally for optimized updates.

```go
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
```

### SessionSaveOpts

Options for the `Save()` method.

```go
type SessionSaveOpts struct {
    IgnoreCallbacks bool // Skip event emission (TC updates and timer resets still apply)
}
```

| Field | Type | Description |
|-------|------|-------------|
| `IgnoreCallbacks` | `bool` | When `true`, skips `EventSessionChanged` emission after saving. TC updates and timer resets for running sessions still apply. Default is `false`. |

## IClientSession Methods

The following methods are available in `IClientSession`.

### ID

Returns the session's ID as an `int64` value.

```go
id := session.ID()
```

### UUID

Returns the session's unique identifier as a `string` value.

```go
uuid := session.UUID()
```

### DeviceID

Returns the database ID of the device that owns this session as an `int64` value. This is useful when you need to look up the device associated with a session.

```go
deviceID := session.DeviceID()

// Use with FindClientById to get the full device
device, err := api.SessionsMgr().FindClientById(ctx, deviceID)
```

### Plugin

Returns the provider plugin of the session record as an `IPluginApi` interface.

```go
plugin := session.Plugin()
```

### Type

Returns the session type. The session type is a `SessionType` value (string type).

```go
t := session.Type()
```

The available session types are:

| Value | Description
| --- | ---
| `"time"` | Represents a `time` session in seconds. Time sessions are sessions that expire when the allocated time is consumed.
| `"data"` | Represents a `data` session in Megabytes. Data sessions are sessions that expire when the allocated data is consumed.
| `"time-or-data"` | Represents a `time-or-data` session. Time or data sessions are sessions that are limited by time or data, whichever is consumed first.

### TimeSecs

Returns the allocated session time in seconds. This is only applicable for `time` and `time_or_data` sessions.

```go
secs := session.TimeSecs()
```

### DataMb

Returns the allocated session data in Megabytes. This is only applicable for `data` and `time_or_data` sessions.

```go
mb := session.DataMb()
```

### TimeConsumption

Returns the consumed session time in seconds. The return type is a `uint` value. This is only applicable for `time` and `time_or_data` sessions. It is used to track the consumed time of the session.

```go
consumedSecs := session.TimeConsumption()
```

### DataConsumption

Returns the consumed session data in Megabytes. The return type is a `float64` value. This is only applicable for `data` and `time_or_data` sessions. It is used to track the consumed data of the session.

```go
consumedMb := session.DataConsumption()
```

### ConsumedTimeSecs

Returns the raw stored time consumption in seconds (without elapsed time calculation). Use this for syncing/persistence where you need the base value without the elapsed time since `resumed_at`.

```go
rawSecs := session.ConsumedTimeSecs()
```

### ConsumedDataMb

Returns the raw stored data consumption in megabytes. Use this for syncing/persistence where you need the base value.

```go
rawMb := session.ConsumedDataMb()
```

### RemainingTime

Returns the remaining session time in seconds. The return type is a `uint` value and is calculated by subtracting the time consumption from the allocated time. This is only applicable for `time` and `time_or_data` sessions.

```go
t := session.RemainingTime()
```

### RemainingData

Returns the remaining session data in Megabytes. The return type is a `float64` value and is calculated by subtracting the data consumption from the allocated data. This is only applicable for `data` and `time_or_data` sessions.

```go
mb := session.RemainingData()
```

### IsConsumed

Returns `true` if the session resources are fully consumed. A session is considered consumed when:

- For `time` sessions: remaining time <= 0
- For `data` sessions: data consumption >= data allowance
- For `time-or-data` sessions: either time or data is exhausted
- For any session type: expiration date has passed

```go
if session.IsConsumed() {
    // Session is no longer usable
}
```

### IsExpired

Returns `true` if the session has passed its expiration date. This only checks the expiration date (calculated from `ExpDays` and `StartedAt`), not whether time/data resources are exhausted. Returns `false` if the session has no expiration date set.

```go
if session.IsExpired() {
    // Session has expired by date
}
```

### IsRunning

Returns `true` if the session is currently active (i.e., `ResumedAt` is not nil). This indicates whether the session is actively tracking time consumption.

```go
if session.IsRunning() {
    // Session is currently active
}
```

### IsAvailable

Returns `true` if the session is available for use. A session is NOT available when any of the following is true:

- `StartedAt` is set, OR
- `ResumedAt` is set, OR
- There's consumption data (`TimeConsumption > 0` or `DataConsumption > 0`), OR
- The session has expired (`IsExpired()` returns true)

This is useful for determining session status in admin interfaces:

```go
if session.IsAvailable() {
    // Session has never been used and not expired - show as "Available"
} else if session.IsConsumed() {
    // Session is fully consumed - show as "Consumed"
} else if session.IsExpired() {
    // Session has expired by date - show as "Expired"
} else if session.IsRunning() {
    // Session is currently active - show as "Active"
} else {
    // Session was started but is not running - show as "Paused"
}
```

### StartedAt

Returns a `*time.Time` value representing the time the session started. A `nil` value is returned if the session has not started.

```go
s := session.StartedAt()
```

### ResumedAt

Returns a `*time.Time` value representing the time the session was last resumed. A `nil` value is returned if the session is not currently running. This is used for calculating elapsed time since the session started running.

```go
resumed := session.ResumedAt()
```

### CreatedAt

Returns a `time.Time` value representing the time the session was created.

```go
d := session.CreatedAt()
```

### UpdatedAt

Returns a `time.Time` value representing the time the session was last updated.

```go
updated := session.UpdatedAt()
```

### ExpDays

Returns a `*int` value representing the expiration days after the session is started. A `nil` value is returned if the session does not have expiration date.

```go
exp := session.ExpDays()
```

### ExpiresAt

Returns a `*time.Time` value representing the time the session will expire. The expiration time is calculated based on the session start time and the expiration days. A `nil` value is returned if the session does not have expiration date.

```go
expAt := session.ExpiresAt()
```

### DownMbits

Returns the download speed of the session in Megabits per second (mbps).

```go
mbps := session.DownMbits()
```

### UpMbits

Returns the upload speed of the session in Megabits per second (mbps).

```go
mbps := session.UpMbits()
```

### UseGlobalSpeed

Returns a `bool` value indicating if the session uses the global [bandwidth settings](./config-api.md#bandwidth) for the network interface which the [IClientDevice](./client-device.md) is connected.

```go
useGlobal := session.UseGlobalSpeed()
```

### Data

Returns a snapshot of all session data fields as a `SessionData` struct with pre-computed values. This method acquires the mutex once and returns all fields, reducing lock contention compared to calling individual getters. The `TimeCons` field includes elapsed time for running sessions.

```go
data := session.Data()
fmt.Printf("Session: %s - Remaining: %d secs\n", data.UUID, data.RemainingTime)

// Use pre-computed status fields
if data.IsAvailable {
    fmt.Println("Session is available")
} else if data.IsConsumed {
    fmt.Println("Session is consumed")
}
```

The `SessionData` struct contains raw fields and pre-computed values:

**Raw Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `int64` | Session database ID |
| `UUID` | `string` | Session UUID |
| `DeviceID` | `int64` | Device database ID |
| `Type` | `SessionType` | Session type (time/data/time-or-data) |
| `TimeSecs` | `int` | Allocated time in seconds |
| `DataMb` | `float64` | Allocated data in MB |
| `TimeCons` | `int` | Consumed time (includes elapsed for running sessions) |
| `DataCons` | `float64` | Consumed data in MB |
| `DownMbits` | `int` | Download speed limit in Mbps |
| `UpMbits` | `int` | Upload speed limit in Mbps |
| `UseGlobalSpeed` | `bool` | Whether to use global speed settings |
| `ExpDays` | `*int` | Expiration days (nil if no expiration) |
| `StartedAt` | `*time.Time` | Session start time |
| `ResumedAt` | `*time.Time` | Last resume time |
| `CreatedAt` | `time.Time` | Creation timestamp |
| `UpdatedAt` | `time.Time` | Last update timestamp |

**Pre-computed Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `RemainingTime` | `int` | Remaining time in seconds |
| `RemainingData` | `float64` | Remaining data in MB |
| `ExpiresAt` | `*time.Time` | Expiration time (nil if no expiration) |
| `IsExpired` | `bool` | True if session is expired |
| `IsAvailable` | `bool` | True if session has never been started |
| `IsConsumed` | `bool` | True if session is consumed |
| `IsRunning` | `bool` | True if session is running |

!!! note "Pre-computed Values"
    The `SessionData` struct has no methods - all derived values are pre-computed when `Data()` is called. This ensures consistent values within a single snapshot.

### IncTimeCons

Increments the consumed session time by `n` seconds. The new value is not saved until the [save](#save) method is called.

```go
session.IncTimeCons(60)
session.Save(ctx, nil)
```

### IncDataCons

Increments the consumed session data by `n` Megabytes. The new value is not saved until the [save](#save) method is called.

```go
session.IncDataCons(10)
session.Save(ctx, nil)
```

### SetData

Sets multiple session fields in a single batch operation. This is more efficient than calling individual setters when you need to update multiple fields, as it acquires the session lock only once.

**Parameters:**

- `data` (`SessionUpdateData`): Struct containing fields to update. Only non-nil pointer fields will be updated.

**Performance:** This method is **~7x more efficient** than calling individual setters when updating multiple fields, as it reduces lock acquisitions and memory allocations.

**Fields in SessionUpdateData:**

| Field | Type | Description |
|-------|------|-------------|
| `TimeSecs` | `*int` | Allocated time in seconds |
| `DataMb` | `*float64` | Allocated data in megabytes |
| `TimeCons` | `*int` | Time consumption in seconds |
| `DataCons` | `*float64` | Data consumption in megabytes |
| `DownMbits` | `*int` | Download speed limit in Mbps |
| `UpMbits` | `*int` | Upload speed limit in Mbps |
| `UseGlobalSpeed` | `*bool` | Whether to use global speed settings |
| `StartedAt` | `*time.Time` | When session was first started |
| `ResumedAt` | `*time.Time` | When session was last resumed |
| `ExpDays` | `*int` | Expiration days |

**Example:**

```go
import sdkapi "sdk/api"

// Helper functions to create pointers
func intPtr(i int) *int { return &i }
func float64Ptr(f float64) *float64 { return &f }
func boolPtr(b bool) *bool { return &b }

// Update multiple fields efficiently in a single batch operation
session.SetData(sdkapi.SessionUpdateData{
    TimeSecs:       intPtr(3600),
    DataMb:         float64Ptr(1024.0),
    TimeCons:       intPtr(600),
    UseGlobalSpeed: boolPtr(true),
    DownMbits:      intPtr(10),
    UpMbits:        intPtr(5),
})

// Don't forget to save
err := session.Save(ctx, nil)
```

!!! tip "Performance Optimization"
    Use `SetData()` instead of multiple individual setters when updating 3 or more fields. This significantly reduces lock contention in high-throughput scenarios.

### Save

Save the session changes to the database. After saving, side effects are applied for running sessions:

- **Timer reset** if time allocation or consumption changed
- **TC (traffic control) update** if bandwidth settings changed
- **Event emission** (`EventSessionChanged`) unless `IgnoreCallbacks` is set

**Signature:**

```go
func (s *ClientSession) Save(ctx context.Context, opts *SessionSaveOpts) error
```

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Context for the operation |
| `opts` | `*SessionSaveOpts` | Optional settings. Pass `nil` for default behavior. |

**Examples:**

```go
// Default behavior - applies all side effects and emits events
err := session.Save(ctx, nil)

// Skip event emission (useful for batch updates)
err := session.Save(ctx, &sdkapi.SessionSaveOpts{IgnoreCallbacks: true})
```

### Reload

Reload the session data from the database.

```go
err := session.Reload(ctx)
```

### PersistToDB

Saves the session state directly to the database without triggering side effects. Unlike `Save()`, this does NOT apply TC updates, timer resets, or emit events, and does NOT clear dirty flags. This is used for internal bookkeeping operations such as periodic saves and stop operations.

!!! warning "Internal Use"
    This method is primarily intended for internal system operations. For normal session updates, use [Save](#save) instead.

```go
err := session.PersistToDB(ctx)
```

### SnapshotTimeCons

Atomically snapshots elapsed time into stored consumption and resets `resumedAt`. This method does NOT set dirty flags as it's an internal bookkeeping operation.

- If `clearResumed` is `true`, sets `resumedAt` to `nil` (session is stopping)
- If `clearResumed` is `false`, resets `resumedAt` to `now` (checkpoint for continued tracking)

Returns the elapsed seconds for logging purposes.

!!! warning "Internal Use"
    This method is primarily intended for internal system operations. For normal session updates, use the standard consumption methods instead.

```go
elapsed := session.SnapshotTimeCons(true)  // Session stopping
elapsed := session.SnapshotTimeCons(false) // Checkpoint, continue tracking
```

### Sync

Reloads session data from the database and applies any changes to the running session. This is useful when session data has been modified externally (e.g., by another process or direct database update) and you need to synchronize the in-memory state with the database.

For running sessions, `Sync()` will automatically:

- **Reset the timer** if time allocation (`TimeSecs`) or consumption (`TimeCons`) changed
- **Update TC (traffic control) rules** if bandwidth settings (`DownMbits`, `UpMbits`, `UseGlobalSpeed`) changed
- **Stop the session** if resources are now consumed (time/data exhausted)

After syncing, `EventSessionChanged` is emitted if any fields changed.

```go
// Sync a running session after external modification
sessions, _ := api.SessionsMgr().ListRunningSessions()
for _, session := range sessions {
    err := session.Sync(ctx)
    if err != nil {
        log.Printf("Failed to sync session %d: %v", session.ID(), err)
    }
}
```

**Example: Sync after direct database update**

```go
// Scenario: Admin adds time via direct database query
// UPDATE sessions SET time_secs = time_secs + 3600 WHERE id = 123;

// Get the running session and sync to apply the change
runningSession, ok := api.SessionsMgr().RunningSession(device)
if ok {
    // Sync reloads from DB and resets the timer with new time
    err := runningSession.Sync(ctx)
    if err != nil {
        log.Printf("Sync failed: %v", err)
    }
    // Timer is now reset with the additional hour
}
```

**Example: Sync after external updates**

```go
// After external process writes new consumption values to database
for _, session := range updatedSessions {
    // Find if this session is currently running
    runningSessions, _ := api.SessionsMgr().ListRunningSessions()
    for _, rs := range runningSessions {
        if rs.ID() == session.ID() {
            // Sync to apply external updates to running session
            rs.Sync(ctx)
            break
        }
    }
}
```

!!! tip "Sync vs Reload"
    Use `Sync()` when you need the changes to take effect immediately on running sessions (timer reset, bandwidth update). Use `Reload()` when you just need to refresh the in-memory data without applying side effects.
