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

### FindDeviceByUUID

Finds a client device by its globally unique identifier (UUID). This is useful for cloud sync scenarios where the cloud server needs to reference devices by their UUID rather than local database ID.

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

Finds a session by its globally unique identifier (UUID). This is useful for cloud sync scenarios where the cloud server needs to terminate or query sessions by their UUID.

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
- `SessionType SessionType` - the [type of session](./client-session.md#type) to create (`"time"`, `"data"`, or `"time-or-data"`)
- `TimeSecs int` - the duration of the session in seconds, applicable only for `time` and `time-or-data` session types
- `DataMbytes float64` - the data in megabytes, applicable only for `data` and `time-or-data` session types
- `ExpDays *int` - the expiration in days after the session is started
- `DownMbits int` - the download speed of the session in megabits per second (mbps)
- `UpMbits int` - the upload speed of the session in megabits per second (mbps)
- `UseGlobal bool` - whether to use the global download and upload speed limit

Below is an example of how to use the `CreateSession` method:

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)

    params := CreateSessionParams{
        DevId:       clnt.ID(),
        SessionType: "time-or-data",
        TimeSecs:    3600,     // 1 hour
        DataMbytes:  100.0,    // 100 MB
        ExpDays:     &[]int{30}[0], // 30 days
        DownMbits:   5,        // 5 mbps
        UpMbits:     3,        // 3 mbps
        UseGlobal:   false,
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


### OnSessionEvent

Registers a callback function for session events. The callback receives a `SessionEventData` struct containing the session and device information.

Available session events:
- `"session:connected"`
- `"session:disconnected"`
- `"session:expired"`
- `"session:updated"`

```go
api.SessionsMgr().OnSessionEvent("session:connected", func(data SessionEventData) {
    // Handle session connected event
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

## Updating Sessions

To update a session's time, data, or bandwidth, use the [IClientSession](./client-session.md) setter methods followed by [Save](./client-session.md#save). The `Save()` method automatically applies side effects for running sessions (timer reset, TC rule updates) and emits `EventSessionUpdated`.

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

## Cloud Sync Integration

The `ISessionsMgrApi` provides methods that enable cloud synchronization of sessions and devices. This allows you to build plugins that sync local hotspot data to a cloud server and receive remote commands.

### Syncing Events to Cloud

Use the event callbacks to push incremental updates to your cloud server:

```go
func Init(api sdkapi.IPluginApi) error {
    machineID := api.Machine().GetID()
    
    // Sync session events
    api.SessionsMgr().OnSessionEvent(sdkapi.EventSessionConnected, func(data sdkapi.SessionEventData) {
        syncToCloud(machineID, "session_connected", map[string]interface{}{
            "session_uuid": data.Session.UUID(),
            "device_uuid":  data.Device.UUID(),
            "device_id":    data.Session.DeviceID(),
        })
    })
    
    // Sync client events
    api.SessionsMgr().OnClientEvent(sdkapi.EventClientCreated, func(clnt sdkapi.IClientDevice) {
        syncToCloud(machineID, "client_created", map[string]interface{}{
            "device_uuid": clnt.UUID(),
            "mac_addr":    clnt.MacAddr(),
            "hostname":    clnt.Hostname(),
        })
    })
}
```

### Receiving Cloud Commands

Use UUID-based lookups to process commands from your cloud server:

```go
func handleCloudCommand(ctx context.Context, api sdkapi.IPluginApi, cmd CloudCommand) error {
    switch cmd.Action {
    case "terminate_session":
        // Find session by UUID from cloud
        session, err := api.SessionsMgr().FindSessionByUUID(ctx, cmd.SessionUUID)
        if err != nil {
            return err
        }
        
        // Get the device and disconnect
        device, err := api.SessionsMgr().FindClientById(ctx, session.DeviceID())
        if err != nil {
            return err
        }
        
        return api.SessionsMgr().Disconnect(ctx, device, "Terminated by cloud")
        
    case "block_device":
        // Find device by UUID from cloud
        device, err := api.SessionsMgr().FindDeviceByUUID(ctx, cmd.DeviceUUID)
        if err != nil {
            return err
        }
        
        // Block the device
        return device.Update(ctx, sdkapi.UpdateDeviceParams{
            Mac:      device.MacAddr(),
            Ip:       device.IpAddr(),
            Hostname: device.Hostname(),
            UUID:     device.UUID(),
            Status:   int(sdkapi.Blocked),
        })
    }
    return nil
}
```
