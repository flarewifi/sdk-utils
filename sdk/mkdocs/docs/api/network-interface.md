# INetworkInterface

The `INetworkInterface` interface represents a network interface configuration in the Flarewifi system. It provides methods to access interface information including name, associated device, status, and IP configuration.

Network interfaces are obtained through the [INetworkApi](./network-api.md) interface.

## INetworkInterface Methods

The following methods are available in `INetworkInterface`:

### Ifname

Returns the name of the network interface (e.g., "lan", "wan", "br-lan").

```go
iface, _ := api.Network().GetInterface("br-lan")
name := iface.Ifname()
fmt.Printf("Interface name: %s\n", name) // "br-lan"
```

### Device

Returns the network device associated with this interface.

```go
device, err := iface.Device()
if err != nil {
    // handle error
}

fmt.Printf("Associated device: %s (%s)\n", device.Name(), device.Type())
```

### Up

Returns `true` if the network interface is up and operational, `false` otherwise.

```go
isUp := iface.Up()
if isUp {
    fmt.Println("Interface is operational")
} else {
    fmt.Println("Interface is down")
}
```

### IpV4Addr

Returns the IPv4 address configuration of the network interface.

```go
ipv4, err := iface.IpV4Addr()
if err != nil {
    // handle error
}

if ipv4 != nil {
    fmt.Printf("IP Address: %s\n", ipv4.Addr)
    fmt.Printf("Netmask: /%d\n", ipv4.Netmask)
}
```

### IPNet

Returns the IP network information of the interface as a `*net.IPNet`.

```go
ipNet, err := iface.IPNet()
if err != nil {
    // handle error
}

if ipNet != nil {
    fmt.Printf("Network: %s\n", ipNet.String())
}
```

## Types

### NetworkIpv4

The `NetworkIpv4` struct represents IPv4 configuration:

```go
type NetworkIpv4 struct {
    Addr    string // IP address (e.g., "192.168.1.1")
    Netmask int    // Netmask in CIDR notation (e.g., 24 for /24)
}
```

## Usage Example

```go
// Get all network interfaces
interfaces, err := api.Network().ListInterfaces()
if err != nil {
    // handle error
}

// Iterate through interfaces and print information
for _, iface := range interfaces {
    fmt.Printf("Interface: %s\n", iface.Ifname())
    fmt.Printf("  Status: %s\n", map[bool]string{true: "Up", false: "Down"}[iface.Up()])

    // Get associated device
    device, err := iface.Device()
    if err == nil {
        fmt.Printf("  Device: %s (%s)\n", device.Name(), device.Type())
    }

    // Get IP configuration
    ipv4, err := iface.IpV4Addr()
    if err == nil && ipv4 != nil {
        fmt.Printf("  IP: %s/%d\n", ipv4.Addr, ipv4.Netmask)
    }

    // Get network information
    ipNet, err := iface.IPNet()
    if err == nil && ipNet != nil {
        fmt.Printf("  Network: %s\n", ipNet.String())
    }

    fmt.Println()
}
```

## Finding Client Interface

You can find which interface a client is connected to using their IP address:

```go
clientIP := "192.168.1.100"
iface, err := api.Network().FindByIp(clientIP)
if err != nil {
    // handle error
}

fmt.Printf("Client %s is connected to interface: %s\n", clientIP, iface.Ifname())
```</content>
<parameter name="filePath">/Users/adonesp/Projects/flarehotspot/sdk/mkdocs/docs/api/network-interface.md