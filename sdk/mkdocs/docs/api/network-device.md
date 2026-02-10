# INetworkDevice

The `INetworkDevice` interface represents a network device (such as Ethernet, WLAN, bridge, or VLAN interfaces) in the Flare Hotspot system. It provides methods to access device information including name, type, MAC address, status, and traffic statistics.

Network devices are obtained through the [INetworkApi](./network-api.md) interface.

## INetworkDevice Methods

The following methods are available in `INetworkDevice`:

### Name

Returns the name of the network device (e.g., "eth0", "wlan0", "br-lan").

```go
device, _ := api.Network().GetDevice("eth0")
name := device.Name()
fmt.Printf("Device name: %s\n", name) // "eth0"
```

### Type

Returns the type of the network device as a `NetDevType`.

```go
deviceType := device.Type()
fmt.Printf("Device type: %s\n", deviceType)
```

Available device types:

| Value | Description |
|-------|-------------|
| `"bridge"` | `NetDevBridge` - Bridge device |
| `"ethernet"` | `NetDevEther` - Ethernet device |
| `"wlan"` | `NetDevWLAN` - Wireless LAN device |
| `"vlan"` | `NetDevVLAN` - VLAN device |

### MacAddr

Returns the MAC address of the network device.

```go
mac := device.MacAddr()
fmt.Printf("MAC address: %s\n", mac) // "00:11:22:33:44:55"
```

### Up

Returns `true` if the network device is up and operational, `false` otherwise.

```go
isUp := device.Up()
if isUp {
    fmt.Println("Device is operational")
} else {
    fmt.Println("Device is down")
}
```

### SpeedMbps

Returns the link speed of the network device in Mbps (megabits per second). Returns 1000 Mbps as a fallback if the speed cannot be detected or parsed.

```go
speed := device.SpeedMbps()
fmt.Printf("Link speed: %d Mbps\n", speed) // e.g., "Link speed: 1000 Mbps"
```

This method automatically parses the underlying link speed and handles various formats (e.g., "1000M", "10G"). When the bandwidth configuration has upload/download speed set to 0, the system uses this auto-detected link speed as the global speed limit.

### BridgeMembers

Returns the names of bridge member ports. This is only applicable for bridge devices.

```go
members := device.BridgeMembers()
fmt.Printf("Bridge members: %v\n", members) // ["eth0", "eth1"]
```

### RxBytes

Returns the current receive (RX) bytes count of the network device.

```go
rxBytes := device.RxBytes()
fmt.Printf("Received bytes: %d\n", rxBytes)
```

### TxBytes

Returns the current transmit (TX) bytes count of the network device.

```go
txBytes := device.TxBytes()
fmt.Printf("Transmitted bytes: %d\n", txBytes)
```

## Usage Example

```go
// Get all network devices
devices, err := api.Network().ListDevices()
if err != nil {
    // handle error
}

// Iterate through devices and print information
for _, device := range devices {
    fmt.Printf("Device: %s\n", device.Name())
    fmt.Printf("  Type: %s\n", device.Type())
    fmt.Printf("  MAC: %s\n", device.MacAddr())
    fmt.Printf("  Status: %s\n", map[bool]string{true: "Up", false: "Down"}[device.Up()])
    fmt.Printf("  Speed: %d Mbps\n", device.SpeedMbps())
    fmt.Printf("  RX: %d bytes, TX: %d bytes\n", device.RxBytes(), device.TxBytes())

    if device.Type() == sdkapi.NetDevBridge {
        fmt.Printf("  Bridge members: %v\n", device.BridgeMembers())
    }
    fmt.Println()
}
```</content>
<parameter name="filePath">/Users/adonesp/Projects/flarehotspot/sdk/mkdocs/docs/api/network-device.md