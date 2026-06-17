# INetworkApi

The `INetworkApi` provides methods to access and manage network devices and interfaces in the Flarewifi system. It allows you to retrieve information about network hardware, interfaces, and traffic data.

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

fmt.Printf("Client is connected to interface: %s\n", iface.Ifname())
```

### GetWanInterface

Returns the WAN network interface.

- **Production (OpenWRT):** Searches for standard WAN interface names in priority order: "wan", "wan6", "wan0"
- **Development:** Returns the container's default network interface (the one used to reach external networks)

```go
wanIface, err := api.Network().GetWanInterface()
if err != nil {
    api.Logger().Error("No WAN interface found: " + err.Error())
    return
}

// Get WAN device for traffic rates
wanDevice, err := wanIface.Device()
if err != nil {
    return
}

// Get real-time traffic rates
downloadRate := wanDevice.RxRate() // bytes per second
uploadRate := wanDevice.TxRate()   // bytes per second
fmt.Printf("WAN Download: %.1f KB/s\n", float64(downloadRate)/1024)
fmt.Printf("WAN Upload: %.1f KB/s\n", float64(uploadRate)/1024)

// Check physical link status
if wanDevice.Carrier() {
    fmt.Printf("WAN Link: %d Mbps %s-duplex\n", wanDevice.SpeedMbps(), wanDevice.Duplex())
} else {
    fmt.Println("WAN Link: No carrier (cable unplugged?)")
}
```

### Traffic

Returns the network traffic API for monitoring bandwidth usage.

```go
trafficAPI := api.Network().Traffic()
fmt.Println(trafficAPI) // ITrafficApi
```

### OnReady

Registers a callback that will be called when the network API is ready to use. This is useful for plugins that need to perform network-related operations during initialization.

If the network is already ready when `OnReady` is called, the callback will be executed immediately (synchronously). Otherwise, the callback will be queued and executed after the network initialization completes.

```go
api.Network().OnReady(func() {
    // Network is now ready
    devices, err := api.Network().ListDevices()
    if err != nil {
        api.Logger().Error("Failed to list devices: " + err.Error())
        return
    }
    
    for _, device := range devices {
        api.Logger().Info("Found device: " + device.Name())
    }
})
```

**Note:** Callbacks are executed synchronously in the order they were registered. If a callback panics, the error will be logged and execution will continue with the next callback.

## Related Interfaces

- [INetworkDevice](./network-device.md) - Represents a network device
- [INetworkInterface](./network-interface.md) - Represents a network interface
- [ITrafficApi](./network-traffic.md) - Provides network traffic monitoring</content>
<parameter name="filePath">/Users/adonesp/Projects/flarehotspot/sdk/mkdocs/docs/api/network-api.md