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
    ip := "10.0.0.25"
    clnt, err := api.SessionsMgr().FindClientByIp(r.Context(), ip)
    if err != nil {
        // handle error - device not found
    }
    
    // Access device information
    mac := clnt.MacAddr()
    hostname := clnt.Hostname()
    fmt.Printf("Device at %s has MAC %s (hostname: %s)\n", ip, mac, hostname)
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
    isConnected, err = api.SessionsMgr().IsConnected(clnt)
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
        
        fmt.Printf("Device %s (%s) - Session %d: %d secs remaining\n",
            device.MacAddr(), device.IpAddr(), session.ID(), session.RemainingTime())
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
    summary, err = api.SessionsMgr().SessionSummary(r.Context(), clnt)
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
| `IpAddress` | `string` | Device IP address |
| `Hostname` | `string` | Device hostname |
| `Status` | `DeviceStatus` | Device status: `DeviceStatusConnected`, `DeviceStatusDisconnected`, or `DeviceStatusBlocked` |
| `CreatedAt` | `time.Time` | When the device was created |
| `UpdatedAt` | `time.Time` | When the device was last updated |

```go
// Example: Wrap device data for use with SDK methods
func wrapDeviceData(data DeviceRow) sdkapi.IClientDevice {
    return api.SessionsMgr().NewClientDevice(sdkapi.NewDeviceParams{
        ID:         data.ID,
        UUID:       data.UUID,
        MacAddress: data.Mac,
        IpAddress:  data.IP,
        Hostname:   data.Hostname,
        Status:     sdkapi.DeviceStatus(data.Status),
        CreatedAt:  data.CreatedAt,
        UpdatedAt:  data.UpdatedAt,
    })
}

// Example: Create device from external data and emit event
func processDeviceData(deviceData ExternalDeviceData) {
    device := api.SessionsMgr().NewClientDevice(sdkapi.NewDeviceParams{
        ID:         deviceData.ID,
        UUID:       deviceData.UUID,
        MacAddress: deviceData.Mac,
        IpAddress:  deviceData.IP,
        Hostname:   deviceData.Hostname,
        Status:     sdkapi.DeviceStatusConnected,
        CreatedAt:  deviceData.CreatedAt,
        UpdatedAt:  time.Now(),
    })
    
    // Use SDK methods on the wrapped device
    device.Emit("device:processed", []byte(`{"status": "ok"}`))
}
```

### OnSessionEvent

Registers a callback function for session events. The callback receives a `SessionEventData` struct containing the session and device information.

The event system enables plugins to react to session lifecycle changes in real-time, allowing for features like analytics, notifications, audit logging, and external integrations.

#### Available Session Events

**`sdkapi.EventSessionCreated` - "session:created"**

Fires when a new session is created via `CreateSession()`.

**Use cases:** Track session inventory, initialize session-related resources, sync new sessions to external systems.

```go
api.SessionsMgr().OnSessionEvent(sdkapi.EventSessionCreated, func(data sdkapi.SessionEventData) error {
    session := data.Session
    device := data.Device
    
    log.Printf("New session created: %s for device %s", session.UUID(), device.MacAddr())
    return nil
})
```

**`sdkapi.EventSessionConnected` - "session:connected"**

Fires when a device connects to the internet and starts consuming a session.

**Use cases:** Send welcome notifications, log connection events, start session monitoring.

```go
api.SessionsMgr().OnSessionEvent(sdkapi.EventSessionConnected, func(data sdkapi.SessionEventData) error {
    session := data.Session
    device := data.Device
    
    log.Printf("Device %s connected with session %d", device.MacAddr(), session.ID())
    return nil
})
```

**`sdkapi.EventSessionDisconnected` - "session:disconnected"**

Fires when a device disconnects from the internet. The session is paused but not consumed.

**Use cases:** Send disconnect notifications, save consumption state, calculate session duration.

```go
api.SessionsMgr().OnSessionEvent(sdkapi.EventSessionDisconnected, func(data sdkapi.SessionEventData) error {
    session := data.Session
    device := data.Device
    
    log.Printf("Device %s disconnected, consumed %d seconds", 
        device.MacAddr(), session.ConsumedTimeSecs())
    return nil
})
```

**`sdkapi.EventSessionConsumed` - "session:expired"**

Fires when a session is fully consumed (time or data exhausted).

**Use cases:** Trigger upsell flows, cleanup resources, send expiry notifications.

```go
api.SessionsMgr().OnSessionEvent(sdkapi.EventSessionConsumed, func(data sdkapi.SessionEventData) error {
    session := data.Session
    device := data.Device
    
    log.Printf("Session %d fully consumed for device %s", session.ID(), device.MacAddr())
    
    // Send notification to device
    device.Emit("session:expired", []byte("Your session has expired. Please purchase more time."))
    return nil
})
```

**`sdkapi.EventSessionChanged` - "session:changed"** ⭐

Fires when session data is externally modified via `session.Save()` (time, data, bandwidth, consumption, etc.).

**Important:** This event is NOT emitted during internal state transitions (session start/stop/periodic saves). It only fires when plugins or users explicitly modify a session via `session.Save()`.

**Use cases:** External system synchronization, audit logs of user modifications, database replication.

See [EventSessionChanged Deep Dive](#eventsessionchanged-deep-dive) below for detailed documentation.

**`sdkapi.EventSessionDeleted` - "session:deleted"**

Fires when a session is permanently deleted via `DeleteSession()`.

**Use cases:** Cleanup related records, sync deletions to external systems, audit trail.

```go
api.SessionsMgr().OnSessionEvent(sdkapi.EventSessionDeleted, func(data sdkapi.SessionEventData) error {
    session := data.Session
    device := data.Device
    
    log.Printf("Session %d deleted for device %s", session.ID(), device.MacAddr())
    return nil
})
```

#### EventSessionChanged Deep Dive

The `EventSessionChanged` event fires only when a session is explicitly modified via `session.Save()` after using setter methods. This event represents user or plugin-initiated changes, not internal state transitions.

##### When EventSessionChanged Fires

When you modify a session using setter methods and call `Save()`, the event fires with precise change tracking via `SessionChangedFields`.

**What triggers it:**

- Time allocation changes via `SetTimeSecs()` + `Save()`
- Data allocation changes via `SetDataMb()` + `Save()`
- Bandwidth changes via `SetDownMbits()`, `SetUpMbits()`, or `SetUseGlobalSpeed()` + `Save()`
- Expiration changes via `SetExpDays()` + `Save()`
- Consumption updates (e.g., from external sync or manual adjustments) + `Save()`

**What does NOT trigger it:**

- Session start (`Start()` uses `PersistToDB()` internally, not `Save()`)
- Session stop (`Stop()` uses `PersistToDB()` internally, not `Save()`)
- Periodic saves (every 60 seconds for crash protection, uses `PersistToDB()`)
- Session chaining (when a consumed session is replaced by the next available session)
- Cloud synchronization (syncing session data from cloud uses `PersistToDB()` to avoid duplicate events)

**Automatic side effects for running sessions:**

When `Save()` is called on a running session, the system automatically:

- **Resets the timer** if time allocation or consumption changed
- **Updates TC (traffic control) rules** if bandwidth settings changed
- **Checks if session is consumed** and stops it if resources are exhausted

**Example: Admin adds time to a user's session**

```go
func addTimeToSession(ctx context.Context, sessionID int64, additionalSecs int) error {
    session, err := api.SessionsMgr().FindSessionByID(ctx, sessionID)
    if err != nil {
        return err
    }
    
    // Add time to existing allocation
    newTimeSecs := session.TimeSecs() + additionalSecs
    session.SetTimeSecs(newTimeSecs)
    
    // Save triggers EventSessionChanged
    // If session is running, timer is automatically reset
    err = session.Save(ctx)
    if err != nil {
        return err
    }
    
    log.Printf("Added %d seconds to session %d (now %d total)", 
        additionalSecs, sessionID, newTimeSecs)
    return nil
}
```

**Change tracking:** The `ChangedFields` field in `SessionEventData` indicates exactly which fields were modified:

```go
api.SessionsMgr().OnSessionEvent(sdkapi.EventSessionChanged, func(data sdkapi.SessionEventData) error {
    session := data.Session
    changed := data.ChangedFields
    
    // Check what changed
    if changed.TimeSecs || changed.TimeCons {
        log.Printf("Session %d time changed - Remaining: %d secs", 
            session.ID(), session.RemainingTime())
    }
    if changed.DataMb || changed.DataCons {
        log.Printf("Session %d data changed - Remaining: %.2f MB", 
            session.ID(), session.RemainingData())
    }
    if changed.DownMbits || changed.UpMbits || changed.UseGlobalSpeed {
        log.Printf("Session %d bandwidth changed - Down: %d, Up: %d Mbps", 
            session.ID(), session.DownMbits(), session.UpMbits())
    }
    
    return nil
})
```

#### Save() vs PersistToDB()

The SDK provides two methods for persisting session changes to the database, each serving a different purpose:

##### session.Save(ctx)

**Use for:** User or plugin-initiated modifications that should trigger `EventSessionChanged`.

**Behavior:**
- Emits `EventSessionChanged` event if session data actually changed
- Clears dirty flags after successful save
- Triggers automatic side effects for running sessions (timer reset, TC rule updates)
- Use this when an external actor (admin, API, plugin) modifies a session

**Example:**
```go
// Admin adds time to a session
session.SetTimeSecs(session.TimeSecs() + 3600)
session.Save(ctx) // ✅ Emits EventSessionChanged
```

##### session.PersistToDB(ctx)

**Use for:** Internal bookkeeping operations that should NOT trigger events.

**Behavior:**
- Does NOT emit `EventSessionChanged` event
- Does NOT clear dirty flags (used for periodic snapshots)
- Does NOT trigger automatic side effects
- Use this for internal operations like periodic saves or state transitions

**Example:**
```go
// Cloud sync updates session from remote data
session.SetTimeCons(cloudSession.TimeConsumption)
session.SetDataCons(cloudSession.DataConsumption)
session.PersistToDB(ctx) // ✅ No event emitted (prevents duplicate sync)
```

**When to use each:**

| Scenario | Method | Reason |
|----------|--------|--------|
| Admin modifies session via UI | `Save()` | Should trigger sync/audit events |
| API endpoint updates session | `Save()` | Should trigger sync/audit events |
| Plugin adds time to session | `Save()` | Should trigger sync/audit events |
| Cloud sync fetches remote changes | `PersistToDB()` | Avoid duplicate sync events |
| Session start/stop (internal) | `PersistToDB()` | Internal state transition |
| Periodic save (crash protection) | `PersistToDB()` | Background bookkeeping |
| Session chaining (consumed→next) | `PersistToDB()` | Internal transition |

**Important:** The core WiFi hotspot system uses `PersistToDB()` for internal operations (`Start()`, `Stop()`, periodic saves), which is why `EventSessionChanged` does NOT fire during these operations. This prevents duplicate event emissions and maintains clean separation between user-initiated changes and internal state management.

#### SessionEventData Structure

All session event callbacks receive a `SessionEventData` struct:

```go
type SessionEventData struct {
    Session       IClientSession       // The session that triggered the event
    Device        IClientDevice        // The device that owns the session
    ChangedFields SessionChangedFields // Which fields changed (only set for EventSessionChanged)
}
```

**ChangedFields** is only populated for `EventSessionChanged` events. For other events, all fields will be `false`. See [SessionChangedFields](./client-session.md#sessionchangedfields) for the full list of trackable fields.

**Available methods on `Session`:**

| Method | Return Type | Description |
|--------|-------------|-------------|
| `ID()` | `int64` | Session database ID |
| `UUID()` | `string` | Session unique identifier (globally unique) |
| `DeviceID()` | `int64` | ID of device that owns this session |
| `Type()` | `SessionType` | Session type: `"time"`, `"data"`, or `"time-or-data"` |
| `TimeSecs()` | `int` | Total allocated time in seconds |
| `DataMb()` | `float64` | Total allocated data in megabytes |
| `ConsumedTimeSecs()` | `int` | Time consumed so far (includes elapsed time if running) |
| `ConsumedDataMb()` | `float64` | Data consumed so far (includes active consumption if running) |
| `RemainingTime()` | `int` | Remaining time in seconds (accounts for running time) |
| `RemainingData()` | `float64` | Remaining data in MB (accounts for active consumption) |
| `DownMbits()` | `int` | Download speed limit in Mbps |
| `UpMbits()` | `int` | Upload speed limit in Mbps |
| `UseGlobalSpeed()` | `bool` | Whether using global bandwidth settings |
| `ExpDays()` | `*int` | Expiration days after session start (nil if no expiry) |
| `IsRunning()` | `bool` | True if session is currently active |
| `IsConsumed()` | `bool` | True if resources are fully exhausted |
| `IsExpired()` | `bool` | True if past expiration date |
| `IsAvailable()` | `bool` | True if session has never been started |
| `StartedAt()` | `*time.Time` | When session was first started (nil if never started) |
| `ResumedAt()` | `*time.Time` | When session was last resumed (nil if not running) |
| `CreatedAt()` | `time.Time` | When session was created |
| `UpdatedAt()` | `time.Time` | When session was last updated |

**Available methods on `Device`:**

| Method | Return Type | Description |
|--------|-------------|-------------|
| `ID()` | `int64` | Device database ID |
| `UUID()` | `string` | Device unique identifier (globally unique) |
| `MacAddr()` | `string` | Device MAC address |
| `IpAddr()` | `string` | Device IP address |
| `Hostname()` | `string` | Device hostname |
| `Status()` | `DeviceStatus` | Connection status: connected/disconnected/blocked |
| `IsConnected()` | `bool` | True if device has an active session |
| `Emit()` | - | Send SSE message to device browser |

#### Complete Plugin Examples

##### Example 1: External System Synchronization

Sync session modifications to external systems (databases, APIs, etc.).

```go
func (plugin *SyncPlugin) Init(api sdkapi.IPluginApi) error {
    sessionsMgr := api.SessionsMgr()
    
    // Listen for session changes (user modifications only)
    sessionsMgr.OnSessionEvent(sdkapi.EventSessionChanged, func(data sdkapi.SessionEventData) error {
        session := data.Session
        device := data.Device
        
        // Prepare update payload
        update := SessionUpdate{
            SessionUUID:   session.UUID(),
            DeviceUUID:    device.UUID(),
            TimeSecs:      session.TimeSecs(),
            DataMb:        session.DataMb(),
            TimeCons:      session.ConsumedTimeSecs(),
            DataCons:      session.ConsumedDataMb(),
            DownMbits:     session.DownMbits(),
            UpMbits:       session.UpMbits(),
            IsActive:      session.IsRunning(),
            UpdatedAt:     time.Now(),
        }
        
        // Sync to external system (async to avoid blocking)
        go plugin.syncToExternalSystem(update)
        
        return nil
    })
    
    return nil
}
```

##### Example 2: Audit Logging

Create comprehensive audit trail of all session changes.

```go
func (plugin *AuditPlugin) Init(api sdkapi.IPluginApi) error {
    sessionsMgr := api.SessionsMgr()
    
    sessionsMgr.OnSessionEvent(sdkapi.EventSessionChanged, func(data sdkapi.SessionEventData) error {
        session := data.Session
        device := data.Device
        
        // Create comprehensive audit log entry
        auditEntry := AuditLogEntry{
            EventType:     "session:changed",
            SessionID:     session.ID(),
            SessionUUID:   session.UUID(),
            DeviceID:      device.ID(),
            DeviceUUID:    device.UUID(),
            DeviceMAC:     device.MacAddr(),
            DeviceIP:      device.IpAddr(),
            SessionType:   string(session.Type()),
            TimeSecs:      session.TimeSecs(),
            TimeConsumed:  session.ConsumedTimeSecs(),
            TimeRemaining: session.RemainingTime(),
            DataMb:        session.DataMb(),
            DataConsumed:  session.ConsumedDataMb(),
            DataRemaining: session.RemainingData(),
            DownMbits:     session.DownMbits(),
            UpMbits:       session.UpMbits(),
            IsRunning:     session.IsRunning(),
            IsConsumed:    session.IsConsumed(),
            IsExpired:     session.IsExpired(),
            Timestamp:     time.Now(),
        }
        
        // Persist to audit log database (async)
        go plugin.db.CreateAuditLog(auditEntry)
        
        return nil
    })
    
    return nil
}
```

#### Performance Considerations

**Event frequency:**

- `EventSessionChanged` fires only when explicitly triggered via `session.Save()` after modifications
- Not emitted during internal operations (start, stop, periodic saves, chaining)
- Typical frequency: Low (only on user/admin actions like bandwidth changes, time/data additions)

**Best practices:**

1. **Keep callbacks fast** - Avoid expensive computations or blocking operations in the callback
2. **Use goroutines for I/O** - Network calls, database writes, and file operations should be async
3. **Handle errors gracefully** - Return errors only for critical failures that should halt event processing

**Example: Efficient event handler**

```go
sessionsMgr.OnSessionEvent(sdkapi.EventSessionChanged, func(data sdkapi.SessionEventData) error {
    session := data.Session
    device := data.Device
    
    // FAST: Prepare lightweight data structure
    update := SessionUpdate{
        SessionUUID: session.UUID(),
        DeviceUUID:  device.UUID(),
        TimeSecs:    session.TimeSecs(),
        DataMb:      session.DataMb(),
        IsActive:    session.IsRunning(),
        Timestamp:   time.Now(),
    }
    
    // ASYNC: Process heavy operations in background
    go plugin.syncToExternalSystem(update)
    
    // Return immediately (callback completes in <1ms)
    return nil
})
```

**Anti-patterns to avoid:**

```go
// ❌ BAD: Blocking database write
sessionsMgr.OnSessionEvent(sdkapi.EventSessionChanged, func(data sdkapi.SessionEventData) error {
    // This blocks ALL session updates for 50-200ms per event!
    return plugin.db.SaveSessionSnapshot(data.Session)
})

// ❌ BAD: Synchronous HTTP call
sessionsMgr.OnSessionEvent(sdkapi.EventSessionChanged, func(data sdkapi.SessionEventData) error {
    // This blocks for network latency (100ms-2s per event!)
    return plugin.sendToAPI(data.Session)
})

// ✅ GOOD: Async processing
sessionsMgr.OnSessionEvent(sdkapi.EventSessionChanged, func(data sdkapi.SessionEventData) error {
    // Fast callback, heavy work in background
    go plugin.db.SaveSessionSnapshot(data.Session)
    go plugin.sendToAPI(data.Session)
    return nil // Returns immediately
})
```

### OnClientEvent

Registers a callback function for client device events.

Available client events:
- `"client:created"`
- `"client:updated"`
- `"client:connected"`
- `"client:disconnected"`

```go
api.SessionsMgr().OnClientEvent("client:connected", func(clnt IClientDevice) {
    // Handle client connected event
})
```

### ListSessions

Returns a paginated list of sessions with optional search and filters. This is useful for building admin interfaces that display and filter sessions.

The `ListSessionsParams` struct contains:

| Field | Type | Description |
|-------|------|-------------|
| `Search` | `*string` | Search by session UUID, device UUID/MAC/hostname/IP, provider package, or voucher code |
| `DeviceID` | `*int64` | Filter sessions for a specific device |
| `Availability` | `*SessionFilterAvailability` | Filter by availability: `"available"`, `"consumed"`, or `"expired"` (nil = all) |
| `SessionType` | `*SessionType` | Filter by session type: `"time"`, `"data"`, or `"time-or-data"` (nil = all) |
| `DateStart` | `*time.Time` | Sessions created on or after this date |
| `DateEnd` | `*time.Time` | Sessions created on or before this date |
| `TimeSecsGt` | `*int` | Sessions with time_secs greater than this value |
| `TimeSecsLt` | `*int` | Sessions with time_secs less than this value |
| `DataMbGt` | `*float64` | Sessions with data_mbytes greater than this value |
| `DataMbLt` | `*float64` | Sessions with data_mbytes less than this value |
| `Page` | `int` | Page number (1-indexed) |
| `PerPage` | `int` | Number of sessions per page |

The `SessionFilterAvailability` constants:

- `SessionFilterAvailable` - Sessions with remaining time/data that are not expired
- `SessionFilterConsumed` - Sessions that are fully consumed (time/data exhausted)
- `SessionFilterExpired` - Sessions that have passed their expiration date

```go
func (w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // List all available sessions, page 1
    result, err := api.SessionsMgr().ListSessions(ctx, sdkapi.ListSessionsParams{
        Availability: &sdkapi.SessionFilterAvailable,
        Page:         1,
        PerPage:      20,
    })
    if err != nil {
        // handle error
    }
    
    fmt.Printf("Found %d sessions (total: %d)\n", len(result.Sessions), result.Count)
    for _, session := range result.Sessions {
        fmt.Printf("Session %d: %s, remaining time: %d secs\n", 
            session.ID(), session.Type(), session.RemainingTime())
    }
}
```

**Example: List sessions for a specific device with date filtering**

```go
func (w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    deviceID := int64(123)
    
    // Filter sessions created in the last 7 days
    now := time.Now()
    weekAgo := now.AddDate(0, 0, -7)
    
    result, err := api.SessionsMgr().ListSessions(ctx, sdkapi.ListSessionsParams{
        DeviceID:  &deviceID,
        DateStart: &weekAgo,
        DateEnd:   &now,
        Page:      1,
        PerPage:   50,
    })
    if err != nil {
        // handle error
    }
    
    // Process sessions...
}
```

**Example: Search sessions by MAC address**

```go
func (w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    searchTerm := "AA:BB:CC"
    
    result, err := api.SessionsMgr().ListSessions(ctx, sdkapi.ListSessionsParams{
        Search:  &searchTerm,
        Page:    1,
        PerPage: 20,
    })
    if err != nil {
        // handle error
    }
    
    // Process matching sessions...
}
```

## Updating Sessions

To update a session's time, data, or bandwidth, use the [IClientSession](./client-session.md) setter methods followed by [Save](./client-session.md#save). The `Save()` method automatically applies side effects for running sessions (timer reset, TC rule updates) and emits `EventSessionUpdated`.

For detailed information about the `EventSessionUpdated` event, including when it fires and how to use it in plugins, see [EventSessionUpdated Deep Dive](#eventsessionupdated-deep-dive).

### Update Remaining Time

```go
session, err := api.SessionsMgr().FindSessionByID(r.Context(), sessionID)
if err != nil {
    // handle error
}

// Set new total time = desired remaining + already consumed
newTimeSecs := remainingSecs + session.ConsumedTimeSecs()
session.SetTimeSecs(newTimeSecs)
err = session.Save(r.Context())
```

### Update Remaining Data

```go
session, err := api.SessionsMgr().FindSessionByID(r.Context(), sessionID)
if err != nil {
    // handle error
}

// Set new total data = desired remaining + already consumed
newDataMb := remainingMb + session.ConsumedDataMb()
session.SetDataMb(newDataMb)
err = session.Save(r.Context())
```

### Update Bandwidth

```go
session, err := api.SessionsMgr().FindSessionByID(r.Context(), sessionID)
if err != nil {
    // handle error
}

session.SetDownMbits(10) // 10 Mbps download
session.SetUpMbits(5)    // 5 Mbps upload
session.SetUseGlobalSpeed(false)
err = session.Save(r.Context())
```
