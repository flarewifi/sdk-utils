# ISessionsMgrApi

The `ISessionsMgrApi` contains methods to manage the client device [sessions](./client-session.md).

## ISessionsMgrApi Methods

### FindClientById

Finds a client device by its database ID. It takes a [context](https://gobyexample.com/context) and the device ID as parameters.

```go
func (w http.ResponseWriter, r *http.Request) {
    devId := int64(123)
    clnt, err := api.SessionsMgr().FindClientById(r.Context(), devId)
    if err != nil {
        // handle error
    }
}
```

### FindClientByMac

Finds a client device by its MAC address. This is useful when you have a MAC address from network operations (e.g., ARP, DHCP) and need to find the associated device.

```go
func (w http.ResponseWriter, r *http.Request) {
    mac := "AA:BB:CC:DD:EE:FF"
    clnt, err := api.SessionsMgr().FindClientByMac(r.Context(), mac)
    if err != nil {
        // handle error - device not found
    }
    // Use the device...
}
```

### FindClientByIp

Finds a client device by its IP address. This is useful when you have an IP address (e.g., from an HTTP request or network discovery) and need to find the associated device.

```go
func (w http.ResponseWriter, r *http.Request) {
    ip := "10.0.0.25" // can be an IPv4 or IPv6 address
    clnt, err := api.SessionsMgr().FindClientByIp(r.Context(), ip)
    if err != nil {
        // handle error - device not found
    }
    
    // Access device information
    mac := clnt.MacAddr()
    hostname := clnt.Hostname()
    fmt.Printf("Device MAC %s (hostname: %s) IPv4=%s IPv6=%s\n",
        mac, hostname, clnt.Ipv4Addr(), clnt.Ipv6Addr())
}
```

### FindDeviceByUUID

Finds a client device by its globally unique identifier (UUID). This is useful when you need to reference devices by their UUID rather than local database ID.

```go
func (w http.ResponseWriter, r *http.Request) {
    uuid := "550e8400-e29b-41d4-a716-446655440000"
    clnt, err := api.SessionsMgr().FindDeviceByUUID(r.Context(), uuid)
    if err != nil {
        // handle error - device not found
    }
    // Use the device...
}
```

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
    device, _ := api.SessionsMgr().FindClientById(r.Context(), deviceID)
    
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
        device, err := api.SessionsMgr().FindClientById(ctx, session.DeviceID())
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

Wraps device data into an [IClientDevice](./client-device.md) object without performing additional database queries. This is useful when you already have device data from queries (e.g., batch operations) and want to use SDK methods like `Update()`, `Emit()`, and `Subscribe()`.

The `NewDeviceParams` struct contains all device fields:

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `int64` | Device database ID |
| `UUID` | `string` | Device unique identifier |
| `MacAddress` | `string` | Device MAC address |
| `Ipv4Address` | `string` | Device IPv4 address (empty string if device has no IPv4) |
| `Ipv6Address` | `string` | Device IPv6 address (empty string if device has no IPv6) |
| `Hostname` | `string` | Device hostname |
| `Status` | `DeviceStatus` | Device status: `DeviceStatusConnected`, `DeviceStatusDisconnected`, or `DeviceStatusBlocked` |
| `CreatedAt` | `time.Time` | When the device was created |
| `UpdatedAt` | `time.Time` | When the device was last updated |

```go
// Example: Wrap device data for use with SDK methods
func wrapDeviceData(data DeviceRow) sdkapi.IClientDevice {
    return api.SessionsMgr().NewClientDevice(sdkapi.NewDeviceParams{
        ID:          data.ID,
        UUID:        data.UUID,
        MacAddress:  data.Mac,
        Ipv4Address: data.Ipv4,  // empty string if device has no IPv4
        Ipv6Address: data.Ipv6,  // empty string if device has no IPv6
        Hostname:    data.Hostname,
        Status:      sdkapi.DeviceStatus(data.Status),
        CreatedAt:   data.CreatedAt,
        UpdatedAt:   data.UpdatedAt,
    })
}

// Example: Create device from external data and emit event
func processDeviceData(deviceData ExternalDeviceData) {
    device := api.SessionsMgr().NewClientDevice(sdkapi.NewDeviceParams{
        ID:          deviceData.ID,
        UUID:        deviceData.UUID,
        MacAddress:  deviceData.Mac,
        Ipv4Address: deviceData.Ipv4,
        Ipv6Address: deviceData.Ipv6,
        Hostname:    deviceData.Hostname,
        Status:      sdkapi.DeviceStatusConnected,
        CreatedAt:   deviceData.CreatedAt,
        UpdatedAt:   time.Now(),
    })
    
    // Use SDK methods on the wrapped device
    device.Emit("device:processed", []byte(`{"status": "ok"}`))
}
```

### FindRunningSessionByUUID

Finds a currently running session by its UUID. Returns the session and `true` if found, or `nil` and `false` if no running session exists with the given UUID. Unlike `FindSessionByUUID` which queries the database, this method only checks in-memory running sessions for better performance when you only need to know if a session is actively connected.

```go
session, ok := api.SessionsMgr().FindRunningSessionByUUID("660e8400-e29b-41d4-a716-446655440001")
if ok {
    fmt.Printf("Session %d is running, %d secs remaining\n", session.ID(), session.RemainingTime())
}
```

### MergeClientDevices

Merges the source device into the target device. All sessions, purchases, and fingerprints are transferred from source to target. The source device is deleted after the merge.

Active sessions on either device are disconnected before the merge. If the target device had an active session it is reconnected afterward. Before any data is transferred it emits [`EventClientBeforeMerge`](./events-api.md#onclientbeforemerge) (via `OnClientBeforeMerge`) with **both** devices — a subscriber returning an error cancels the merge and it is returned here. After a successful merge it emits `OnClientMerge` (`EventClientMergeData`) so plugins can notify external systems.

```go
func handleMergeDevices(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    targetID := int64(10)
    sourceID := int64(20)

    err := api.SessionsMgr().MergeClientDevices(ctx, targetID, sourceID)
    if err != nil {
        // handle error - source device is deleted on success
    }
}
```

The merge event carries an `EventClientMergeData` struct:

```go
type EventClientMergeData struct {
    Target           IClientDevice // The surviving device (before and after the merge)
    Source           IClientDevice // The device about to be deleted — set only for the
                                   // pre-merge EventClientBeforeMerge; nil for EventClientMerge
    SourceDeviceID   int64         // DB ID of the deleted source device
    SourceDeviceUUID string        // UUID of the deleted source device (captured before deletion)
}
```

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
