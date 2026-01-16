# IClientSession

The `IClientSession` represents a session for the [IClientDevice](./client-device.md). It can be manipulated using the [SessionsMgrApi](./sessions-mgr-api.md). To get an instance of `IClientSession`, you can use the [CreateSession](./sessions-mgr-api.md#createsession), [RunningSession](./sessions-mgr-api.md#runningsession) or [AvailableSession](./sessions-mgr-api.md#availablesession) methods from [SessionMgrApi](./sessions-mgr-api.md). For example:

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)
    session, _ := api.SessionsMgr().RunningSession(clnt)
}
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

Returns the database ID of the device that owns this session as an `int64` value. This is useful when you need to look up the device associated with a session, particularly in cloud sync scenarios.

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

### StartedAt

Returns a `*time.Time` value representing the time the session started. A `nil` value is returned if the session has not started.

```go
s := session.StartedAt()
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
session.SetStartedAt(time.Now())
session.Save()
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
