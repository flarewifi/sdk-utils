# IClientsMgrApi

The `IClientsMgrApi` manages [client devices](./client-device.md): looking them up, wrapping preview objects, and registering a client device a plugin already knows the exact MAC/IP/hostname for — for example, importing wifi client history from an external source. `RegisterClient` differs from the live captive-portal registration flow used internally by the core: there is no cookie/fingerprint/ARP-NDP disambiguation, since the caller already knows exactly which MAC/IP/hostname to register.

## IClientsMgrApi Methods

### FindClientById

Finds a client device by its database ID. It takes a [context](https://gobyexample.com/context) and the device ID as parameters.

```go
func (w http.ResponseWriter, r *http.Request) {
    devId := int64(123)
    clnt, err := api.ClientsMgr().FindClientById(r.Context(), devId)
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
    clnt, err := api.ClientsMgr().FindClientByMac(r.Context(), mac)
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
    clnt, err := api.ClientsMgr().FindClientByIp(r.Context(), ip)
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

### FindClientByUUID

Finds a client device by its globally unique identifier (UUID). This is useful when you need to reference devices by their UUID rather than local database ID.

```go
func (w http.ResponseWriter, r *http.Request) {
    uuid := "550e8400-e29b-41d4-a716-446655440000"
    clnt, err := api.ClientsMgr().FindClientByUUID(r.Context(), uuid)
    if err != nil {
        // handle error - device not found
    }
    // Use the device...
}
```

### NewClientDevice

Wraps device data into an [IClientDevice](./client-device.md) object without performing additional database queries. This is useful when you already have device data from queries (e.g., batch operations) and want to use SDK methods like `Update()`, `Emit()`, and `Subscribe()`. Also use this to build an in-memory preview (e.g. `ID` left at `0`) to pass to [`RegisterClient`](#registerclient).

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
    return api.ClientsMgr().NewClientDevice(sdkapi.NewDeviceParams{
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
    device := api.ClientsMgr().NewClientDevice(sdkapi.NewDeviceParams{
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

### RegisterClient

Persists a client device preview — built via [`NewClientDevice`](#newclientdevice) — as a real device record. It emits the same `EventClientBeforeCreate`, `EventClientCreated`, and `EventClientRegistered` events the live captive-portal registration flow emits, so other plugins observing those events (whitelist checks, notifications, etc.) see this registration the same way.

Returns `sdkapi.ErrClientAlreadyRegistered` if a device with the given MAC address is already registered — check for this specifically with `errors.Is` before treating a failure as an expected, ignorable duplicate. Any other error (an `EventClientBeforeCreate` subscriber vetoing the registration, a database error, etc.) is a real failure and should not be silently treated the same way.

```go
func importClient(api sdkapi.IPluginApi, mac, ipv4, hostname string) error {
    preview := api.ClientsMgr().NewClientDevice(sdkapi.NewDeviceParams{
        MacAddress:  mac,
        Ipv4Address: ipv4,
        Hostname:    hostname,
        Status:      sdkapi.DeviceStatusDisconnected,
    })

    if err := api.ClientsMgr().RegisterClient(preview); err != nil {
        if errors.Is(err, sdkapi.ErrClientAlreadyRegistered) {
            // Expected — this MAC is already a known device. Not an error.
            return nil
        }
        return err // a real failure — log/report it, don't swallow it
    }

    // Look up the persisted record if you need it (RegisterClient itself
    // returns only an error, matching the live registration flow).
    clnt, err := api.ClientsMgr().FindClientByMac(context.Background(), mac)
    return err
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

    err := api.ClientsMgr().MergeClientDevices(ctx, targetID, sourceID)
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
