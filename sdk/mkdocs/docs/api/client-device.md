# IClientDevice

The `IClientDevice` represents a client device/host connected in your network and is possibly accessing the captive portal using a browser.
It can be retrieved using the [Http.GetClientDevice](./http-api.md#getclientdevice) method in your [handler](../guides/routes-and-links.md#handlerfunc).

```go title="main.go"
// http handler
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)
    fmt.Println(clnt) // IClientDevice
}
```

The `clnt` variable is an instance of the `IClientDevice` interface.

## 1. IClientDevice Methods {#clientdevice-methods}

Below are the methods available on the `IClientDevice` instance.

### Id

Returns the database `pgtype.UUID` of the client device.

```go
uuid := clnt.Id()
```

### Hostname

Returns a `string` value of the client device hostname.

```go
h := clnt.Hostname()
```

### IpAddr

Returns a `string` value of the client device IP address.

```go
ip := clnt.IpAddr()
```

### MacAddr

Returns a `string` value of the client device MAC address.

```go
mac := clnt.MacAddr()
```

### Update

Updates the client device record in the database.

```go
func (w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    clnt, _ := api.Http().GetClientDevice(r)
    newMac := "00:11:22:33:44:55"
    newIp := "192.168.1.123"
    newHostname := "new-hostname"

    if err := clnt.Update(ctx, newMac, newIp, newHostname); err != nil {
        // handle error
    }
}
```

### Emit

Emits an [event](#events) to the client device.

```go
data := map[string]interface{}{"key": "value"}
clnt.Emit("some_event", data)
```

The `data` parameter can any JSON serializable value.

### Subscribe

Used to subscribe to an [event](#events) on the client device. It returns a channel that emits a JSON representation in bytes.

```go
ch := clnt.Subscribe("some_event")

go func() {
    clnt.Emit("some_event", map[string]interface{}{"key": "value"})
}()

bytes := <-ch

var data map[string]interface{}

err := json.Unmarshal(bytes, &data)

fmt.Println(data) // map[key:value]
```

### Unsubscribe

Used to unsubscribe from an [event](#events) on the client device.

```go
// subscribe first
ch := clnt.Subscribe("some_event")

// listen to the channel...

// then unsubscribe
clnt.Unsubscribe("some_event", ch)
```

## 2. Events {#events}

Events are emitted to the client device in the browser.

You can emit an event to a user account using the [ClientDevice.Emit](#emit) method like so:

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().Helpers().GetClientDevice(r)
    evt := "some_event"
    data := map[string]interface{}{"key": "value"}
    clnt.Emit(evt, data)
}
```

You can listen to this events in the browser using the [$flare.events](./flare-variable.md#flare-events) like so:

```js
$flare.events.on("some_event", function(res) {
    console.log("An event occured: ", res.data);
});
```

You can also listen to events in your Go applications using the [ClientDevice.Subscribe](#subscribe) method.

The avaiable events geenerated by the system are:

| Event | Description
|--|--
| `session:connected` | Emitted when a client device is connected to the internet.
| `session:disconnected` | Emitted when a client device is disconnected from the internet.
