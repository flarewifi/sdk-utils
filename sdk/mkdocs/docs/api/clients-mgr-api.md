# IClientsMgrApi

The `IClientsMgrApi` lets a plugin register a [client device](./client-device.md) it already knows the exact MAC/IP/hostname for — for example, importing wifi client history from an external source. It differs from the live captive-portal registration flow used internally by the core: there is no cookie/fingerprint/ARP-NDP disambiguation, since the caller already knows exactly which MAC/IP/hostname to register.

## IClientsMgrApi Methods

### RegisterClient

Persists a client device preview — built via [`SessionsMgr().NewClientDevice`](./sessions-mgr-api.md#newclientdevice) — as a real device record. It emits the same `EventClientBeforeCreate`, `EventClientCreated`, and `EventClientRegistered` events the live captive-portal registration flow emits, so other plugins observing those events (whitelist checks, notifications, etc.) see this registration the same way.

Returns `sdkapi.ErrClientAlreadyRegistered` if a device with the given MAC address is already registered — check for this specifically with `errors.Is` before treating a failure as an expected, ignorable duplicate. Any other error (an `EventClientBeforeCreate` subscriber vetoing the registration, a database error, etc.) is a real failure and should not be silently treated the same way.

```go
func importClient(api sdkapi.IPluginApi, mac, ipv4, hostname string) error {
    preview := api.SessionsMgr().NewClientDevice(sdkapi.NewDeviceParams{
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
    clnt, err := api.SessionsMgr().FindClientByMac(context.Background(), mac)
    return err
}
```
