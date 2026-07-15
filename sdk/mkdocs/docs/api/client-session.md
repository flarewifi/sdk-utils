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

### CreatedAt

Returns a `time.Time` value representing the time the session was created.

```go
d := session.CreatedAt()
```

### Data

Returns a snapshot of all session data fields as a `SessionData` struct with pre-computed values. This method acquires the mutex once and returns all fields, reducing lock contention compared to calling individual getters. The `TimeCons` field includes elapsed time for running sessions (unless the counter is paused by `Pause()`).

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
| `PausedAt` | `*time.Time` | When counters were paused (nil if not paused) |
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
| `IsPaused` | `bool` | True if the time/data counters are paused |

!!! note "Pre-computed Values"
    The `SessionData` struct has no methods - all derived values are pre-computed when `Data()` is called. This ensures consistent values within a single snapshot.

### DataConsumption

Returns the consumed session data in Megabytes. The return type is a `float64` value. This is only applicable for `data` and `time_or_data` sessions. It is used to track the consumed data of the session.

```go
consumedMb := session.DataConsumption()
```

### DataMb

Returns the allocated session data in Megabytes. This is only applicable for `data` and `time_or_data` sessions.

```go
mb := session.DataMb()
```

### DeviceID

Returns the database ID of the device that owns this session as an `int64` value. This is useful when you need to look up the device associated with a session.

```go
deviceID := session.DeviceID()

// Use with FindClientById to get the full device
device, err := api.SessionsMgr().FindClientById(ctx, deviceID)
```

### DownMbits

Returns the download speed of the session in Megabits per second (mbps).

```go
mbps := session.DownMbits()
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

### ID

Returns the session's ID as an `int64` value.

```go
id := session.ID()
```

### IncDataCons

Increments the consumed session data by `n` Megabytes. The new value is not saved until the [save](#save) method is called.

```go
session.IncDataCons(10)
session.Save(ctx, nil)
```

### IncTimeCons

Increments the consumed session time by `n` seconds. The new value is not saved until the [save](#save) method is called.

```go
session.IncTimeCons(60)
session.Save(ctx, nil)
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

### IsPaused

Returns `true` if the time/data counters are paused (`Pause()` was called and `Resume()` has not been called since). **A paused session stays connected — the WiFi client is NOT disconnected** (firewall rules and bandwidth limits remain active); only time and data accounting is frozen.

```go
if session.IsPaused() {
    fmt.Println("Counters are paused (client still connected)")
} else {
    fmt.Println("Time and data are being counted")
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

Returns `true` if the session is currently active (i.e., `ResumedAt` is not nil). This indicates whether the session is connected. A session can be running but have its counters paused — use `IsPaused()` to check if time and data accounting is frozen.

```go
if session.IsRunning() {
    // Session is connected (may be counting or paused)
    if session.IsPaused() {
        // Counters are paused (Pause was called) — client is STILL connected
    } else {
        // Time and data are being counted
    }
}
```

### PersistToDB

Saves the session state directly to the database without triggering side effects. Unlike `Save()`, this does NOT apply TC updates, timer resets, or emit events, and does NOT clear dirty flags. This is used for internal bookkeeping operations such as periodic saves and stop operations.

!!! warning "Internal Use"
    This method is primarily intended for internal system operations. For normal session updates, use [Save](#save) instead.

```go
err := session.PersistToDB(ctx)
```

### Plugin

Returns the provider plugin of the session record as an `IPluginApi` interface.

```go
plugin := session.Plugin()
```

### RawData

Returns a snapshot of raw session data fields as stored in the database (as a `SessionRawData` struct). Unlike `Data()`, `TimeCons` does NOT include elapsed time — it is the exact value stored in the database. Use this for syncing or persistence where you need the base value without elapsed-time adjustment.

```go
raw := session.RawData()
fmt.Printf("Stored time cons: %d secs (no elapsed adjustment)\n", raw.TimeCons)
```

The `SessionRawData` struct contains:

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `int64` | Session database ID |
| `UUID` | `string` | Session UUID |
| `DeviceID` | `int64` | Device database ID |
| `Type` | `SessionType` | Session type (time/data/time-or-data) |
| `TimeSecs` | `int` | Allocated time in seconds |
| `DataMb` | `float64` | Allocated data in MB |
| `TimeCons` | `int` | Raw stored time consumption (no elapsed adjustment) |
| `DataCons` | `float64` | Raw stored data consumption in MB |
| `DownMbits` | `int` | Download speed limit in Mbps |
| `UpMbits` | `int` | Upload speed limit in Mbps |
| `UseGlobalSpeed` | `bool` | Whether to use global speed settings |
| `ExpDays` | `*int` | Expiration days (nil if no expiration) |
| `StartedAt` | `*time.Time` | Session start time |
| `ResumedAt` | `*time.Time` | Last resume time (nil if not running) |
| `CreatedAt` | `time.Time` | Creation timestamp |
| `UpdatedAt` | `time.Time` | Last update timestamp |

### RemainingData

Returns the remaining session data in Megabytes. The return type is a `float64` value and is calculated by subtracting the data consumption from the allocated data. This is only applicable for `data` and `time_or_data` sessions.

```go
mb := session.RemainingData()
```

### RemainingTime

Returns the remaining session time in seconds. The return type is a `uint` value and is calculated by subtracting the time consumption from the allocated time. This is only applicable for `time` and `time_or_data` sessions.

```go
t := session.RemainingTime()
```

### Resume

Resumes both time and data counters after they were paused by `Pause()`. Clears `pausedAt` and resets `resumedAt` to now so elapsed time calculation starts fresh from this point. Data consumption will be counted again from this point forward. (A paused client was never disconnected, so no reconnection is involved — this only un-freezes accounting.)

```go
session.Resume()
err := session.PersistToDB(ctx)
```

### ResumedAt

Returns a `*time.Time` value representing the time the session was last resumed. A `nil` value is returned if the session is not currently running. This is used for calculating elapsed time since the session started running.

```go
resumed := session.ResumedAt()
```

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

!!! tip "Prefer UpdateSession for update+persist"
    For the common "set fields, then save" flow, prefer the atomic [`SessionsMgr().UpdateSession()`](sessions-mgr-api.md#updatesession) instead of `SetData()` + `Save()`.

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

!!! tip "Prefer UpdateSession for update+persist"
    `SetData()` + `Save()` is a two-step sequence — another writer can interleave between the two calls. When you want to apply fields **and** persist them, prefer the atomic [`SessionsMgr().UpdateSession()`](sessions-mgr-api.md#updatesession), which also routes the update to the live in-memory session if it is currently running.

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

!!! note "Reload and Sync were removed"
    Older SDK versions had `Reload()` and `Sync()` methods that re-read the session row from the database into memory. Both were removed: for a running session the in-memory instance is the authoritative copy (it carries unsaved time/data consumption that is only flushed to the database periodically), so overwriting it from the database silently rolled that consumption back. To modify a session, use the atomic [`SessionsMgr().UpdateSession()`](sessions-mgr-api.md#updatesession) — it routes the update to the live instance and applies the same side effects (timer reset, TC update, consumed-check) that `Sync()` used to.

### StartedAt

Returns a `*time.Time` value representing the time the session started. A `nil` value is returned if the session has not started.

```go
s := session.StartedAt()
```

### Status

Returns the session's current status as a `ClientSessionStatus` string. The status is one of:

| Value | Description |
| --- | --- |
| `"running"` | Session is active and counters are counting |
| `"paused"` | Session is still connected but counters are frozen (`Pause()` was called) — the WiFi client is NOT disconnected |
| `"stopped"` | Session is not running (`ResumedAt` is nil) |

```go
switch session.Status() {
case sdkapi.ClientSessionStatusRunning:
    // Active session with counting
case sdkapi.ClientSessionStatusPaused:
    // Active session with paused counters
case sdkapi.ClielntSessionStatusStopped:
    // Session is not running
}
```

### Pause

Pauses both time and data counters by snapshotting elapsed time into stored consumption and setting `pausedAt`. No further time or data is counted until `Resume()` is called.

!!! important "Pause does NOT disconnect the WiFi client"
    The session stays fully connected while paused — the client keeps its internet access, firewall rules and bandwidth (TC) limits remain in place. Pausing only **freezes time/data accounting**; it is not a disconnect. The client's remaining time/data is held constant and resumes exactly where it left off on `Resume()`.

This is useful when you want to temporarily stop time and data tracking without disconnecting the session — for example, to grant a bonus or apply a promotion without the original counters continuing to run in the background, or to stop charging a client whose device has gone idle.

!!! warning "Persist required"
    This method only updates the in-memory state. Call `PersistToDB()` to persist the paused state (`paused_at`) to the database.

```go
session.Pause()
err := session.PersistToDB(ctx)
```

### TimeConsumption

Returns the consumed session time in seconds. The return type is a `uint` value. This is only applicable for `time` and `time_or_data` sessions. It is used to track the consumed time of the session.

```go
consumedSecs := session.TimeConsumption()
```

### TimeSecs

Returns the allocated session time in seconds. This is only applicable for `time` and `time_or_data` sessions.

```go
secs := session.TimeSecs()
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

### UpMbits

Returns the upload speed of the session in Megabits per second (mbps).

```go
mbps := session.UpMbits()
```

### UpdatedAt

Returns a `time.Time` value representing the time the session was last updated.

```go
updated := session.UpdatedAt()
```

### UseGlobalSpeed

Returns a `bool` value indicating if the session uses the global [bandwidth settings](./config-api.md#bandwidth) for the network interface which the [IClientDevice](./client-device.md) is connected.

```go
useGlobal := session.UseGlobalSpeed()
```

### UUID

Returns the session's unique identifier as a `string` value.

```go
uuid := session.UUID()
```
