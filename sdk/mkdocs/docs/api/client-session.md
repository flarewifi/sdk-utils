# IClientSession

The `IClientSession` represents a session for the [IClientDevice](./client-device.md). It can be manipulated using the [SessionsMgrApi](./sessions-mgr-api.md). To get an instance of `IClientSession`, you can use the [CreateSession](./sessions-mgr-api.md#createsession), [CurrSession](./sessions-mgr-api.md#currsession) or [GetSession](./sessions-mgr-api.md#getsession) methods from [SessionMgrApi](./sessions-mgr-api.md). For example:

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)
    session, _ := api.SessionsMgr().CurrSession(clnt)
}
```

## IClientSession Methods

The following methods are available in `IClientSession`.

### Provider

Returns the session [provider](../api/session-provider.md) name. The provider name is a `string` value.

```go
provider := session.Provider()
```

### Type

Returns the session type. The session type is a `uint8` value.

```go
t := session.Type()
```

The available session types are:

| Value | Description
| --- | ---
| 0 | Represents a `time` session in seconds. Time sessions are sessions that expire when the allocated time is consumed.
| 1 | Represents a `data` session in in Megabytes. Data sessions are sessions that expire when the allocated data is consumed.
| 2 | Represents a `time_or_data` session. Time or data sessions are sessions that are limited by time or data, whichever is consumed first.

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

### ExpDays

Returns a `*uint` value representing the expiration days after the session is started. A `nil` value is returned if the session does not have expiration date.

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
session.Save()
```

### Reload

Reload the session data from the database.

```go
session.Reload()
```
