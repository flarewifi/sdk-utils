# INetworkApi

The `INetworkApi` provides methods to access and manage network devices and interfaces in the Flare Hotspot system. It allows you to retrieve information about network hardware, interfaces, and traffic data.

To get an instance of `INetworkApi`:

```go
networkAPI := api.Network()
fmt.Println(networkAPI) // INetworkApi
```

## INetworkApi Methods

The following methods are available in `INetworkApi`:

### ListDevices

Returns a list of all network devices available in the system.

```go
devices, err := api.Network().ListDevices()
if err != nil {
    // handle error
}

for _, device := range devices {
    fmt.Printf("Device: %s, Type: %s\n", device.Name(), device.Type())
}
```

### ListInterfaces

Returns a list of all network interfaces available in the system.

```go
interfaces, err := api.Network().ListInterfaces()
if err != nil {
    // handle error
}

for _, iface := range interfaces {
    fmt.Printf("Interface: %s, IP: %s\n", iface.Name(), iface.IpAddr())
}
```

### GetDevice

Returns data for a specific network device by name.

```go
device, err := api.Network().GetDevice("eth0")
if err != nil {
    // handle error
}

fmt.Printf("Device Name: %s\n", device.Name())
fmt.Printf("Device Type: %s\n", device.Type())
```

### GetInterface

Returns data for a specific network interface by name.

```go
iface, err := api.Network().GetInterface("br-lan")
if err != nil {
    // handle error
}

fmt.Printf("Interface Name: %s\n", iface.Name())
fmt.Printf("IP Address: %s\n", iface.IpAddr())
```

### FindByIp

Returns the network interface that has the specified IP address. This is useful for finding which interface a client is connected to.

```go
iface, err := api.Network().FindByIp("192.168.1.100")
if err != nil {
    // handle error
}

fmt.Printf("Client is connected to interface: %s\n", iface.Name())
```

### Traffic

Returns the network traffic API for monitoring bandwidth usage.

```go
trafficAPI := api.Network().Traffic()
fmt.Println(trafficAPI) // ITrafficApi
```

## Related Interfaces

- [INetworkDevice](./network-device.md) - Represents a network device
- [INetworkInterface](./network-interface.md) - Represents a network interface
- [ITrafficApi](./network-traffic.md) - Provides network traffic monitoring</content>
<parameter name="filePath">/Users/adonesp/Projects/flarehotspot/sdk/mkdocs/docs/api/network-api.md