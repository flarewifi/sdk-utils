# IClientDevice

The `IClientDevice` represents a **client device** — a phone, tablet, laptop/PC, or other end-user host — connecting through the machine's network, possibly accessing the captive portal using a browser. It is **not** the machine itself (the OpenWRT router/hotspot box running this app); see [IMachineApi](./machine-api.md) for machine-level operations.

It can be retrieved using the [Http.GetClientDevice](./http-api.md#getclientdevice) method in your [handler](./http-router-api.md#handler-function).

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

### ID

Returns the database ID of the client device as an `int64`.

```go
id := clnt.ID()
```

### UUID

Returns the UUID of the client device as a `string`.

```go
uuid := clnt.UUID()
```

### CookieToken

Returns the cookie token used for cookie validation. An empty string means no cookie token validation is enforced.

```go
token := clnt.CookieToken()
```

### Hostname

Returns a `string` value of the client device hostname.

```go
h := clnt.Hostname()
```

### Ipv4Addr

Returns the IPv4 address of the device as a `string`. Returns an empty string if the device has no IPv4 address (e.g., IPv6-only device).

```go
ipv4 := clnt.Ipv4Addr()
```

### Ipv6Addr

Returns the IPv6 address of the device as a `string`. Returns an empty string if the device has no IPv6 address.

```go
ipv6 := clnt.Ipv6Addr()
```

### IpAddr

Returns the primary IP address of the device as a `string` for backward compatibility. Returns the IPv4 address if available, otherwise the IPv6 address.

```go
ip := clnt.IpAddr() // IPv4 preferred, fallback to IPv6
```

!!! note "Dual-Stack Devices"
    For dual-stack devices, `IpAddr()` returns only the IPv4 address. Use `Ipv4Addr()` and `Ipv6Addr()` explicitly when you need to handle both protocols, for example when configuring firewall rules or bandwidth shaping.

### MacAddr

Returns a `string` value of the client device MAC address.

```go
mac := clnt.MacAddr()
```

### Status

Returns the status of the client device as a `DeviceStatus` value.

```go
status := clnt.Status()
```

Available device statuses:

| Value | Description
| --- | ---
| `1` | `DeviceStatusConnected` - Device is connected to the internet
| `2` | `DeviceStatusDisconnected` - Device is disconnected from the internet
| `3` | `DeviceStatusBlocked` - Device is blocked from accessing the internet

### CreatedAt

Returns the creation timestamp of the client device as a `time.Time` value.

```go
createdAt := clnt.CreatedAt()
fmt.Println(createdAt.Format("January 02, 2006"))
```

### UpdatedAt

Returns the last update timestamp of the client device as a `time.Time` value.

```go
updatedAt := clnt.UpdatedAt()
fmt.Println(updatedAt.Format("January 02, 2006 3:04 PM"))
```

### Data

Returns a snapshot of all device data fields as a `DeviceData` struct. This method acquires the mutex once and returns all fields, reducing lock contention compared to calling individual getters.

```go
data := clnt.Data()
fmt.Printf("Device: %s (IPv4: %s, IPv6: %s) - Status: %d\n",
    data.MacAddr, data.Ipv4Addr, data.Ipv6Addr, data.Status)
```

The `DeviceData` struct contains:

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `int64` | Database ID |
| `UUID` | `string` | Device UUID |
| `CookieToken` | `string` | Cookie token for validation (empty = no enforcement) |
| `MacAddr` | `string` | MAC address |
| `Ipv4Addr` | `string` | IPv4 address (empty if device has no IPv4) |
| `Ipv6Addr` | `string` | IPv6 address (empty if device has no IPv6) |
| `Hostname` | `string` | Device hostname |
| `Status` | `DeviceStatus` | Device status |
| `IsConnected` | `bool` | True if device has an active internet session |
| `CreatedAt` | `time.Time` | Creation timestamp |
| `UpdatedAt` | `time.Time` | Last update timestamp |

### Update

Updates the client device record in the database using the `UpdateDeviceParams` struct.

The `UpdateDeviceParams` struct contains:

- `UUID string` - the device UUID
- `Mac string` - the new MAC address
- `Ipv4 string` - the new IPv4 address (empty string if device has no IPv4)
- `Ipv6 string` - the new IPv6 address (empty string if device has no IPv6)
- `Hostname string` - the new hostname
- `Status DeviceStatus` - the new device status (`DeviceStatusConnected`, `DeviceStatusDisconnected`, `DeviceStatusBlocked`)

```go
func (w http.ResponseWriter, r *http.Request) {
    clnt, _ := api.Http().GetClientDevice(r)

    params := sdkapi.UpdateDeviceParams{
        UUID:     clnt.UUID(),
        Mac:      "00:11:22:33:44:55",
        Ipv4:     "192.168.1.123",
        Ipv6:     "2001:db8::1",  // empty string if IPv4-only
        Hostname: "new-hostname",
        Status:   sdkapi.DeviceStatusConnected,
    }

    if err := clnt.Update(r.Context(), params); err != nil {
        // handle error
    }
}
```

### Emit

Emits an [event](#events) to the client device.

```go
data := []byte(`{"key": "value"}`)
clnt.Emit("some_event", data)
```

The `data` parameter is a `[]byte` containing JSON data.

### Subscribe

Used to subscribe to an [event](#events) on the client device. It returns a channel that emits a JSON representation in bytes.

```go
ch := clnt.Subscribe("some_event")

go func() {
    clnt.Emit("some_event", []byte(`{"key": "value"}`))
}()
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
    clnt, _ := api.Http().GetClientDevice(r)
    evt := "some_event"
    data := []byte(`{"key": "value"}`)
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
