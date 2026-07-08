# INetworkDevice

The `INetworkDevice` interface represents a network device (such as Ethernet, WLAN, bridge, or VLAN interfaces) in the Flarewifi system. It provides methods to access device information including name, type, MAC address, status, and traffic statistics.

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

Returns `true` if the network device is administratively up, `false` otherwise.

```go
isUp := device.Up()
if isUp {
    fmt.Println("Device is administratively up")
} else {
    fmt.Println("Device is administratively down")
}
```

### Carrier

Returns `true` if the physical link is connected (cable plugged in, signal detected), `false` otherwise. For wireless devices, this indicates association status.

```go
hasCarrier := device.Carrier()
if hasCarrier {
    fmt.Println("Physical link is connected")
} else {
    fmt.Println("No physical link detected")
}
```

**Note:** A device can be administratively `Up()` but have no `Carrier()` if the cable is unplugged.

### SpeedMbps

Returns the link speed of the network device in Mbps (megabits per second). Returns 1000 Mbps as a fallback if the speed cannot be detected or parsed.

```go
speed := device.SpeedMbps()
fmt.Printf("Link speed: %d Mbps\n", speed) // e.g., "Link speed: 1000 Mbps"
```

This method automatically parses the underlying link speed and handles various formats (e.g., "1000M", "10G"). When the bandwidth configuration has upload/download speed set to 0, the system uses this auto-detected link speed as the global speed limit.

### Duplex

Returns the duplex mode of the network device: `"full"`, `"half"`, or `"unknown"`.

```go
duplex := device.Duplex()
fmt.Printf("Duplex mode: %s\n", duplex) // e.g., "full"
```

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

### RxRate

Returns the current download rate in bytes per second. The rate is calculated from the difference in `RxBytes` since the last call to this method.

```go
rxRate := device.RxRate()
fmt.Printf("Download rate: %.1f KB/s\n", float64(rxRate)/1024)
```

**Note:** Returns 0 on the first call since there's no previous reading to compare against.

### TxRate

Returns the current upload rate in bytes per second. The rate is calculated from the difference in `TxBytes` since the last call to this method.

```go
txRate := device.TxRate()
fmt.Printf("Upload rate: %.1f KB/s\n", float64(txRate)/1024)
```

**Note:** Returns 0 on the first call since there's no previous reading to compare against.

### IsBridge

Returns `true` if the network device is a bridge interface.

```go
if device.IsBridge() {
    fmt.Printf("Bridge members: %v\n", device.BridgeMembers())
}
```

### IsVlan

Returns `true` if the network device is a VLAN interface.

```go
if device.IsVlan() {
    fmt.Println("This is a VLAN interface")
}
```

### IsIfb

Returns `true` if the network device is an IFB (Intermediate Functional Block) interface. IFB devices are shadow interfaces used internally for traffic shaping and are identified by a `-ifb` name suffix (e.g. `eth0-ifb`).

```go
if device.IsIfb() {
    fmt.Println("This is an internal traffic-shaping device")
}
```

### IsEthernet

Returns `true` if the network device is an Ethernet interface.

```go
if device.IsEthernet() {
    fmt.Printf("Speed: %d Mbps, Duplex: %s\n", device.SpeedMbps(), device.Duplex())
}
```

### IsWireless

Returns `true` if the network device is a wireless (WLAN) interface.

```go
if device.IsWireless() {
    fmt.Println("This is a wireless interface")
}
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
```
