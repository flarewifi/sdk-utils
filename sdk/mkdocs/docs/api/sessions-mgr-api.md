# ISessionsMgrApi

The `ISessionsMgrApi` contains methods to manage the client device [sessions](./client-session.md).

## ISessionsMgrApi Methods

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

It creates a [IClientSession](./client-session.md) for the [IClientDevice](./client-device.md). It takes the following arguments:

- `context.Context`
- `pgtype.UUID` - the [IClientDevice](./client-device.md) UUID
- `uint8` - the [type of session](./client-session.md#type) to create
- `uint` - the duration of the session in seconds, applicable only for `time` and `time_or_data` session types
- `float64` - the data in mega bytes, applicable only for `data` and `time_or_data` session types
- `*uint` - the expiration in days after the session is started, on top of the duration in seconds
- `int` - the download speed of the session in megabits per second (mbps)
- `int` - the upload speed of the session in megabits per second (mbps)
- `bool` - whether to use the global download and upload speed limit. If `true`, it ignores the download and upload speed arguments

Below is an example of how to use the `CreateSession` method:

```go
func (w http.ResponseWriter, r *http.Request) {
    secs := 60          // 1 minute
    mb := 100.0         // 100 MB
    sessionType := 0    // 0 = time, 1 = data, 2 = time_or_data
    expireDays := 30    // 30 days
    downMbits := 5      // 5 mbps
    uploadMbits := 3    // 3 mbps

    clnt, _ := api.Http().GetClientDevice(r)
    err := api.SessionsMgr().CreateSession(
        r.Context(),
        clnt.Id(),
        sessionType,
        secs,
        mb,
        &expireDays,
        downMbits,
        upMbits,
        false,
    )
}
```

### CurrSession

This is the current running [IClientSession](./client-session.md) of the [IClientDevice](./client-device.md).

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)
    session, ok = api.SessionsMgr().CurrSession(clnt)
}
```

### GetSession

Returns any available [IClientSession](./client-session.md) for the given [IClientDevice](./client-device.md) ID. This may include the current running session or any paused session.

```go
func (w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    clnt, _ := api.Http().GetClientDevice(r)
    session, err = api.SessionsMgr().GetSession(ctx, clnt)
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


### RegisterSessionProvider

Used to register a [session provider](./session-provider.md).
