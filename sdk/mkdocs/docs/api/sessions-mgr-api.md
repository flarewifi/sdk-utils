# ISessionsMgrApi

The `ISessionsMgrApi` contains methods to manage the client device [sessions](./client-session.md).

## ISessionsMgrApi Methods

### FindClientById, FindClientByMac, FindClientByIp, FindDeviceByUUID

!!! warning "Deprecated"
    These four device-lookup methods have moved to [`IClientsMgrApi`](./clients-mgr-api.md) — use `api.ClientsMgr().FindClientById()`, `FindClientByMac()`, `FindClientByIp()`, and `FindClientByUUID()` (renamed from `FindDeviceByUUID`) instead. The versions on `ISessionsMgrApi` are kept only for backward compatibility and delegate to the same underlying implementation.

### FindSessionByUUID

Finds a session by its globally unique identifier (UUID). This is useful when you need to terminate or query sessions by their UUID.

```go
func (w http.ResponseWriter, r *http.Request) {
    sessionUUID := "660e8400-e29b-41d4-a716-446655440001"
    session, err := api.SessionsMgr().FindSessionByUUID(r.Context(), sessionUUID)
    if err != nil {
        // handle error - session not found
    }
    
    // Get the device that owns this session
    deviceID := session.DeviceID()
    device, _ := api.ClientsMgr().FindClientById(r.Context(), deviceID)
    
    // Terminate the session by disconnecting the device
    api.SessionsMgr().Disconnect(r.Context(), device, "Session terminated by cloud")
}
```

### Connect

This method will connect the client device to the internet if the client device has available [IClientSession](./client-session.md) to consume.
It takes a [context](https://gobyexample.com/context), a [IClientDevice](./client-device.md), and a notification `string` as parameters.

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)
    err = api.SessionsMgr().Connect(r.Context(), clnt, "You are now connected to internet.")
}
```

### Disconnect

This method will disconnect the client device from the internet. It will also pause the current running [IClientSession](./client-session.md) of the client device. It takes a [context](https://gobyexample.com/context), a [IClientDevice](./client-device.md) and a notification `string` as parameters.

Before any teardown, it emits [`EventClientBeforeDisconnect`](./events-api.md#onclientevent): if a subscriber returns an error the disconnect is cancelled and the error is returned here. (This fires for explicit `Disconnect()` calls only — automatic session teardown does not.)

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)
    err = api.SessionsMgr().Disconnect(r.Context(), clnt, "You are now disconnected to internet.")
}
```

### IsConnected

Returns `true` if the [IClientDevice](./client-device.md) is connected to the internet, otherwise `false`.

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)
    isConnected := api.SessionsMgr().IsConnected(clnt)
}
```

### CreateSession

It creates a [IClientSession](./client-session.md) for the [IClientDevice](./client-device.md). It takes a `context.Context` and `CreateSessionParams` struct as arguments.

The `CreateSessionParams` struct contains:

- `DevId int64` - the [IClientDevice](./client-device.md) ID
- `Type SessionType` - the [type of session](./client-session.md#type) to create (`"time"`, `"data"`, or `"time-or-data"`)
- `TimeSecs int` - the duration of the session in seconds, applicable only for `time` and `time-or-data` session types
- `DataMb float64` - the data in megabytes, applicable only for `data` and `time-or-data` session types
- `ExpDays *int` - the expiration in days after the session is started
- `DownMbits int` - the download speed of the session in megabits per second (mbps)
- `UpMbits int` - the upload speed of the session in megabits per second (mbps)
- `UseGlobalSpeed bool` - whether to use the global download and upload speed limit
- `TimeCons int` - (optional) initial time consumption in seconds
- `DataCons float64` - (optional) initial data consumption in megabytes

Before the INSERT it emits [`EventSessionBeforeCreate`](./events-api.md#onsessionevent) with an in-memory preview session (`ID == 0`); a subscriber returning an error cancels creation and it is returned here. After success it emits `EventSessionCreated`.

Below is an example of how to use the `CreateSession` method:

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)

    // Helper to create *int pointer
    expDays := 30

    params := sdkapi.CreateSessionParams{
        UUID:           sdkutils.NewUUID(), // Required: generate unique ID
        DevId:          clnt.ID(),
        Type:           sdkapi.SessionTypeTimeOrData,
        TimeSecs:       3600,     // 1 hour
        DataMb:         100.0,    // 100 MB
        ExpDays:        &expDays, // 30 days (use nil for no expiration)
        DownMbits:      5,        // 5 mbps
        UpMbits:        3,        // 3 mbps
        UseGlobalSpeed: false,
    }

    session, err := api.SessionsMgr().CreateSession(r.Context(), params)
}
```

### CreateSessions

Creates a batch of sessions in one call, taking a `context.Context` and a slice of `CreateSessionParams` (same struct as [`CreateSession`](#createsession)). Returns the created [IClientSession](./client-session.md) slice in the same order as the input.

It emits [`EventSessionBatchBeforeCreate`](./events-api.md#onsessionbatchevent) **once** before any DB writes — a subscriber returning an error cancels the whole batch, so no rollback is needed. The single-session `EventSessionBeforeCreate` is **not** fired per item; the batch hook is the cancellation point for bulk creates. Every session in the batch is then inserted inside a single database transaction, so a failure partway through (e.g. a DB error) automatically rolls back every insert made so far — a failed batch never leaves partial rows behind. Only once the transaction commits does it persist any per-session consumption values, emit the per-session `EventSessionCreated` for each session, and finally emit `EventSessionBatchCreated` once with the full list — so no subscriber ever observes a "created" session that a later failure in the same batch then rolls back.

```go
func importSessions(api sdkapi.IPluginApi, devID int64, imports []importedSession) error {
    paramsList := make([]sdkapi.CreateSessionParams, len(imports))
    for i, imp := range imports {
        paramsList[i] = sdkapi.CreateSessionParams{
            UUID:      sdkutils.NewUUID(),
            DevId:     devID,
            Type:      sdkapi.SessionTypeTime,
            TimeSecs:  imp.TimeSecs,
            DownMbits: 5,
            UpMbits:   3,
        }
    }

    sessions, err := api.SessionsMgr().CreateSessions(context.Background(), paramsList)
    return err
}
```

### RunningSession

Returns the current running [IClientSession](./client-session.md) of the [IClientDevice](./client-device.md).

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)
    session, ok := api.SessionsMgr().RunningSession(clnt)
}
```

### ListRunningSessions

Returns all currently active (running) sessions across all devices. These are sessions that are actively connected and consuming time/data. The returned sessions have real-time consumption data (`RemainingTime()` and `RemainingData()` account for elapsed time since the session started).

This is useful for:

- Building admin dashboards showing all active connections
- Monitoring system-wide session usage
- Implementing session management features

```go
func (w http.ResponseWriter, r *http.Request) {
    sessions, err := api.SessionsMgr().ListRunningSessions()
    if err != nil {
        // handle error
    }
    
    fmt.Printf("Currently %d active sessions\n", len(sessions))
    for _, session := range sessions {
        fmt.Printf("Session %d: %s, remaining time: %d secs, remaining data: %.2f MB\n", 
            session.ID(), session.Type(), session.RemainingTime(), session.RemainingData())
    }
}
```

**Example: Display active sessions with device information**

```go
func (w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    sessions, err := api.SessionsMgr().ListRunningSessions()
    if err != nil {
        // handle error
    }
    
    for _, session := range sessions {
        // Get the device for each session
        device, err := api.ClientsMgr().FindClientById(ctx, session.DeviceID())
        if err != nil {
            continue
        }
        
        fmt.Printf("Device %s (IPv4=%s IPv6=%s) - Session %d: %d secs remaining\n",
            device.MacAddr(), device.Ipv4Addr(), device.Ipv6Addr(),
            session.ID(), session.RemainingTime())
    }
}
```

### AvailableSession

Returns any available [IClientSession](./client-session.md) for the given [IClientDevice](./client-device.md). This may include the current running session or any paused session.

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)
    session, err := api.SessionsMgr().AvailableSession(r.Context(), clnt)
}
```

### SessionSummary

Returns the remaining session duration and data for the given [IClientDevice](./client-device.md).

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)
    summary, err := api.SessionsMgr().SessionSummary(r.Context(), clnt)
}
```

### FindSessionByID

Finds a session by its database ID and wraps it into an [IClientSession](./client-session.md) object. This is useful for displaying session information in templates and controllers where you have a session ID from database queries but need access to SDK methods like `RemainingTime()` and `RemainingData()` which account for elapsed time.

```go
func (w http.ResponseWriter, r *http.Request) {
    sessionID := int64(456)
    session, err := api.SessionsMgr().FindSessionByID(r.Context(), sessionID)
    if err != nil {
        // handle error - session not found
    }
    
    // Use SDK methods that calculate elapsed time
    remaining := session.RemainingTime()
    fmt.Printf("Session %d has %d seconds remaining\n", sessionID, remaining)
}
```

### NewClientSession

Wraps session data into an [IClientSession](./client-session.md) object without performing additional database queries. This is useful when you already have session data from queries (e.g., batch operations) and want to use SDK methods like `RemainingTime()` and `RemainingData()` which account for elapsed time.

The `NewClientSessionParams` struct contains all session fields:

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `int64` | Session database ID |
| `UUID` | `string` | Session unique identifier |
| `ProviderPkg` | `string` | Package name of the plugin that created the session |
| `DeviceID` | `int64` | ID of the device that owns this session |
| `Type` | `SessionType` | Type of session: `"time"`, `"data"`, or `"time-or-data"` |
| `TimeSecs` | `int` | Allocated time in seconds |
| `DataMb` | `float64` | Allocated data in megabytes |
| `TimeCons` | `int` | Time consumed in seconds |
| `DataCons` | `float64` | Data consumed in megabytes |
| `StartedAt` | `*time.Time` | When the session was first started |
| `ResumedAt` | `*time.Time` | When the session was last resumed (nil if not running) |
| `ExpDays` | `*int` | Expiration days after session start |
| `DownMbits` | `int` | Download speed limit in Mbps |
| `UpMbits` | `int` | Upload speed limit in Mbps |
| `UseGlobalSpeed` | `bool` | Whether to use global bandwidth settings |
| `CreatedAt` | `time.Time` | When the session was created |
| `UpdatedAt` | `time.Time` | When the session was last updated |

```go
// Example: Wrap session data from external source
func wrapSessionData(data SessionData) sdkapi.IClientSession {
    return api.SessionsMgr().NewClientSession(sdkapi.NewClientSessionParams{
        ID:             data.ID,
        UUID:           data.UUID,
        ProviderPkg:    data.ProviderPkg,
        DeviceID:       data.DeviceID,
        Type:           sdkapi.SessionType(data.Type),
        TimeSecs:       data.TimeSecs,
        DataMb:         data.DataMb,
        TimeCons:       data.ConsumedSecs,
        DataCons:       data.ConsumedMb,
        StartedAt:      data.StartedAt,
        ResumedAt:      data.ResumedAt,
        ExpDays:        data.ExpDays,
        DownMbits:      data.DownMbits,
        UpMbits:        data.UpMbits,
        UseGlobalSpeed: data.UseGlobalSpeed,
        CreatedAt:      data.CreatedAt,
        UpdatedAt:      data.UpdatedAt,
    })
}
```

### NewClientDevice

!!! warning "Deprecated"
    This method has moved to [`IClientsMgrApi`](./clients-mgr-api.md) — use [`api.ClientsMgr().NewClientDevice()`](./clients-mgr-api.md#newclientdevice) instead. The version on `ISessionsMgrApi` is kept only for backward compatibility and delegates to the same underlying implementation.

### FindRunningSessionByUUID

Finds a currently running session by its UUID. Returns the session and `true` if found, or `nil` and `false` if no running session exists with the given UUID. Unlike `FindSessionByUUID` which queries the database, this method only checks in-memory running sessions for better performance when you only need to know if a session is actively connected.

```go
session, ok := api.SessionsMgr().FindRunningSessionByUUID("660e8400-e29b-41d4-a716-446655440001")
if ok {
    fmt.Printf("Session %d is running, %d secs remaining\n", session.ID(), session.RemainingTime())
}
```

### MergeClientDevices

!!! warning "Deprecated"
    This method has moved to [`IClientsMgrApi`](./clients-mgr-api.md) — use [`api.ClientsMgr().MergeClientDevices()`](./clients-mgr-api.md#mergeclientdevices) instead. The version on `ISessionsMgrApi` is kept only for backward compatibility and delegates to the same underlying implementation.

### DeleteSession

Deletes a single session by ID. If the session is currently running, the owning device is disconnected first. Before any disconnect or deletion it emits [`EventSessionBeforeDelete`](./events-api.md#onsessionevent); a subscriber returning an error cancels the deletion and it is returned here. After deletion it emits `EventSessionDeleted`.

```go
err := api.SessionsMgr().DeleteSession(r.Context(), sessionID)
```

### DeleteSessions

Deletes a batch of sessions by ID. It emits [`EventSessionBatchBeforeDelete`](./events-api.md#onsessionbatchevent) **once** before any deletion — a subscriber returning an error cancels the whole batch — then deletes each session, emitting the per-session `EventSessionDeleted`. The single-session `EventSessionBeforeDelete` is **not** fired per item; the batch hook is the cancellation point for bulk deletes. All sessions are resolved up front, so a missing ID fails the batch before anything is removed.

```go
err := api.SessionsMgr().DeleteSessions(r.Context(), []int64{101, 102, 103})
```

### Listing sessions

There is **no `ListSessions` method** on `ISessionsMgrApi`. To list, search, paginate,
or filter sessions, query the core [`sessions`](../guides/database-schema.md#sessions)
table **directly with your plugin's own sqlc queries**, then wrap each row with
[`NewClientSession`](#newclientsession) so `RemainingTime()` / `RemainingData()` and the
computed `Data()` flags (`IsAvailable`, `IsConsumed`, `IsExpired`) still reflect live
consumption.

!!! info "Why query core tables directly?"
    The SDK deliberately keeps its surface small. Listing and filtering are inherently
    app-specific (which columns, which joins, which filters), so they belong in each
    plugin's own SQL rather than a one-size-fits-all API method. Your plugin's `sqlc`
    config already includes the core schema (`../../../../core/resources/migrations`), so
    a query against `sessions` type-checks and generates like any other. See the
    [Core Database Tables](../guides/database-schema.md) guide for the full schema.

**1. Add a query** in your plugin's `resources/queries/sessions.sql`:

```sql
-- name: SearchSessions :many
SELECT s.* FROM sessions s
JOIN devices d ON s.device_id = d.id
LEFT JOIN device_macs dm ON d.id = dm.device_id AND dm.is_current = TRUE
WHERE (@search = '' OR dm.mac_address LIKE '%' || @search || '%'
                   OR d.ipv4_addr   LIKE '%' || @search || '%')
  AND (@device_id = 0 OR s.device_id = @device_id)
ORDER BY s.created_at DESC
LIMIT @row_limit OFFSET @row_offset;
```

**2. Run it and wrap the rows** with `NewClientSession`:

```go
func listSessions(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    db := queries.New(api.SqlDB())

    rows, err := db.SearchSessions(ctx, queries.SearchSessionsParams{
        Search:    "",
        DeviceID:  0,
        RowLimit:  20,
        RowOffset: 0,
    })
    if err != nil {
        // handle error
    }

    for _, row := range rows {
        var expDays *int
        if row.ExpDays.Valid {
            e := int(row.ExpDays.Int64)
            expDays = &e
        }
        session := api.SessionsMgr().NewClientSession(sdkapi.NewClientSessionParams{
            ID:        row.ID,
            UUID:      row.Uuid,
            DeviceID:  row.DeviceID,
            Type:      sdkapi.SessionType(row.SessionType),
            TimeSecs:  int(row.TimeSecs),
            DataMb:    row.DataMbytes,
            TimeCons:  int(row.ConsumptionSecs),
            DataCons:  row.ConsumptionMb,
            ExpDays:   expDays,
            CreatedAt: row.CreatedAt,
        })
        // session.RemainingTime(), session.Data().IsAvailable, etc.
        _ = session
    }
}
```

For **currently running** sessions (live, in-memory), use
[`ListRunningSessions`](#listrunningsessions) instead — those are runtime state, not a
table query.

## Updating Sessions

To update a session's time, data, or bandwidth, use [`session.SetData()`](./client-session.md#setdata) followed by [`session.Save()`](./client-session.md#save). The `Save()` method automatically applies side effects for running sessions (timer reset, TC rule updates) and emits `EventSessionChanged`.

### Update Remaining Time

```go
session, err := api.SessionsMgr().FindSessionByID(r.Context(), sessionID)
if err != nil {
    // handle error
}

// Set new total time = desired remaining + already consumed
newTimeSecs := remainingSecs + session.ConsumedTimeSecs()
session.SetData(sdkapi.SessionUpdateData{TimeSecs: sdkutils.IntPtr(newTimeSecs)})
err = session.Save(r.Context(), nil)
```

### Update Remaining Data

```go
session, err := api.SessionsMgr().FindSessionByID(r.Context(), sessionID)
if err != nil {
    // handle error
}

// Set new total data = desired remaining + already consumed
newDataMb := remainingMb + session.ConsumedDataMb()
session.SetData(sdkapi.SessionUpdateData{DataMb: sdkutils.Float64Ptr(newDataMb)})
err = session.Save(r.Context(), nil)
```

### Update Bandwidth

```go
session, err := api.SessionsMgr().FindSessionByID(r.Context(), sessionID)
if err != nil {
    // handle error
}

session.SetData(sdkapi.SessionUpdateData{
    DownMbits:      sdkutils.IntPtr(10),
    UpMbits:        sdkutils.IntPtr(5),
    UseGlobalSpeed: sdkutils.BoolPtr(false),
})
err = session.Save(r.Context(), nil)
```
