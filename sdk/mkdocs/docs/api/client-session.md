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

Tracks which session fields were modified since the last save. Used internally for optimized updates.

```go
type SessionChangedFields struct {
    Time      bool // timeSecs or timeCons changed
    Data      bool // dataMb or dataCons changed
    Bandwidth bool // downMbits, upMbits, or useGlobal changed
}
```

### SessionSaveParams

Contains parameters passed to the session save callback.

```go
type SessionSaveParams struct {
    Ctx           context.Context
    Session       IClientSession
    ChangedFields SessionChangedFields
}
```

### SessionSaveCallback

A callback function type that is called after a session is saved. This allows the `SessionsMgr` to update running sessions (reset timers, update traffic control rules) and emit events when `session.Save()` is called.

```go
type SessionSaveCallback func(params SessionSaveParams) error
```

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

Returns a snapshot of all session data fields as a `SessionData` struct. This method acquires the mutex once and returns all fields, reducing lock contention compared to calling individual getters. The `TimeCons` field includes elapsed time for running sessions.

```go
data := session.Data()
fmt.Printf("Session: %s - Time: %d/%d secs\n", data.UUID, data.TimeCons, data.TimeSecs)
```

The `SessionData` struct contains:

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

The `SessionData` struct also provides helper methods:

- `RemainingTime() int` - Returns remaining time in seconds
- `RemainingData() float64` - Returns remaining data in MB
- `ExpiresAt() *time.Time` - Returns expiration time
- `IsExpired() bool` - Returns true if session is expired
- `IsConsumed() bool` - Returns true if session is consumed
- `IsRunning() bool` - Returns true if session is running

### RawData

Returns a snapshot of all session data fields with raw stored values. Unlike `Data()`, the `TimeCons` field does NOT include elapsed time calculation. Use this for syncing/persistence where you need the base values.

```go
rawData := session.RawData()
// rawData.TimeCons contains only the stored value, not elapsed time
```

### IncTimeCons

Increments the consumed session time by `n` seconds. The new value is not saved until the [save](#save) method is called.

```go
session.IncTimeCons(60)
session.Save()
```

### IncDataCons

Increments the consumed session data by `n` Megabytes. The new value is not saved until the [save](#save) method is called.

```go
session.IncDataCons(10)
session.Save()
```

### SetTimeSecs

Sets the allocated session time in seconds. This is only applicable for `time` and `time_or_data` sessions. The new value is not saved until the [save](#save) method is called.

```go
session.SetTimeSecs(3600)
err := session.Save()
```

### SetDataMb

Sets the allocated session data in Megabytes. This is only applicable for `data` and `time_or_data` sessions. The new value is not saved until the [save](#save) method is called.

```go
session.SetDataMb(1024)
session.Save()
```

### SetTimeCons

Sets the consumed session time in seconds. This is only applicable for `time` and `time_or_data` sessions. The new value is not saved until the [save](#save) method is called.

```go
session.SetTimeCons(1800)
session.Save()
```

### SetDataCons

Sets the consumed session data in Megabytes. This is only applicable for `data` and `time_or_data` sessions. The new value is not saved until the [save](#save) method is called.

```go
session.SetDataCons(512)
session.Save()
```

### SetStartedAt

Sets the session start time. The new value is not saved until the [save](#save) method is called.

```go
now := time.Now()
session.SetStartedAt(&now)
session.Save(ctx)
```

### SetResumedAt

Sets the time when the session was last resumed. This is used to track when a session starts running. The new value is not saved until the [save](#save) method is called.

```go
now := time.Now()
session.SetResumedAt(&now)
session.Save(ctx)
```

### SetExpDays

Sets the expiration days after the session is started. The new value is not saved until the [save](#save) method is called.

```go
session.SetExpDays(30)
session.Save()
```

### SetDownMbits

Sets the download speed of the session in Megabits per second (mbps). The new value is not saved until the [save](#save) method is called.

```go
session.SetDownMbits(10)
session.Save()
```

### SetUpMbits

Sets the upload speed of the session in Megabits per second (mbps). The new value is not saved until the [save](#save) method is called.

```go
session.SetUpMbits(2)
session.Save()
```

### SetUseGlobalSpeed

Sets a `bool` value indicating if the session uses the global [bandwidth settings](./config-api.md#bandwidth) for the network interface which the [IClientDevice](./client-device.md) is connected. The new value is not saved until the [save](#save) method is called.

```go
session.SetUseGlobalSpeed(true)
session.Save()
```

### Save

Save the session changes to the database.

```go
err := session.Save(ctx)
```

### Reload

Reload the session data from the database.

```go
err := session.Reload(ctx)
```

### PersistToDB

Saves the session state directly to the database without triggering save callbacks. Unlike `Save()`, this does NOT trigger the `onSave` callback and does NOT clear dirty flags. This is used for internal bookkeeping operations such as periodic saves and stop operations.

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
