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
