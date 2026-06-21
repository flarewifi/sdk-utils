# ITrafficApi

The `ITrafficApi` provides real-time network traffic monitoring. It emits traffic data every 5 seconds, allowing plugins to track per-client bandwidth usage.

Access `ITrafficApi` through `INetworkApi`:

```go
trafficApi := api.Network().Traffic()
```

---

## ITrafficApi Methods

### Listen

Returns a channel that receives `TrafficData` every 5 seconds.

```go
func Init(api sdkapi.IPluginApi) error {
    ch := api.Network().Traffic().Listen()

    go func() {
        for data := range ch {
            for ip, stat := range data.Download {
                fmt.Printf("Client %s downloaded %d bytes (%d packets)\n",
                    ip, stat.Bytes, stat.Packets)
            }
        }
    }()

    return nil
}
```

---

## Types

### TrafficData

Represents a snapshot of network traffic data emitted every 5 seconds.

```go
type TrafficData struct {
    Download map[string]ClientStat // Keyed by client IP address
    Upload   map[string]ClientStat // Keyed by client MAC address
}
```

| Field | Key Type | Description |
|-------|----------|-------------|
| `Download` | Client IP address | Download statistics per client |
| `Upload` | Client MAC address | Upload statistics per client |

### ClientStat

Represents traffic statistics for a single client.

```go
type ClientStat struct {
    Packets uint // Number of packets transferred
    Bytes   uint // Number of bytes transferred
}
```

| Field | Description |
|-------|-------------|
| `Packets` | Count of network packets |
| `Bytes` | Total bytes transferred |

---

## Usage Example

### Monitoring Bandwidth Per Client

```go
func Init(api sdkapi.IPluginApi) error {
    go func() {
        ch := api.Network().Traffic().Listen()
        for data := range ch {
            // Download: keyed by client IP
            for ip, stat := range data.Download {
                mbps := float64(stat.Bytes) * 8 / 1_000_000 / 5 // per 5s window
                api.Logger().Info("Download %s: %.2f Mbps", ip, mbps)
            }

            // Upload: keyed by client MAC
            for mac, stat := range data.Upload {
                mbps := float64(stat.Bytes) * 8 / 1_000_000 / 5
                api.Logger().Info("Upload %s: %.2f Mbps", mac, mbps)
            }
        }
    }()

    return nil
}
```

---

## Related

- [INetworkApi](./network-api.md) - Parent API; access `Traffic()` from here
- [INetworkDevice](./network-device.md) - Per-device RxRate/TxRate for real-time rates
