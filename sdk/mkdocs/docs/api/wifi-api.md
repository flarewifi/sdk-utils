# IWifiApi

The `IWifiApi` provides access to WiFi client connection events. It allows plugins to receive notifications when WiFi clients connect to or disconnect from the access point.

## Accessing IWifiApi

```go
wifiApi := api.Wifi()
```

---

## IWifiApi Methods

### OnWifiClientEvent

Registers a callback to be called when a WiFi client connects or disconnects from the access point.

```go
func Init(api sdkapi.IPluginApi) error {
    // Listen for client connections
    api.Wifi().OnWifiClientEvent(sdkapi.WifiEventClientConnected, func(client sdkapi.IWifiClient) {
        api.Logger().Info("WiFi client connected: %s", client.MacAddress())
    })
    
    // Listen for client disconnections
    api.Wifi().OnWifiClientEvent(sdkapi.WifiEventClientDisconnected, func(client sdkapi.IWifiClient) {
        api.Logger().Info("WiFi client disconnected: %s", client.MacAddress())
    })
    
    return nil
}
```

---

## Types

### IWifiClient

The `IWifiClient` interface represents a WiFi client that has connected or disconnected.

| Method | Return Type | Description |
|--------|-------------|-------------|
| `MacAddress()` | `string` | The MAC address of the WiFi client (e.g., `"3e:8d:af:0d:72:bf"`) |

### WifiClientEvent

The `WifiClientEvent` type represents WiFi client events:

```go
type WifiClientEvent string

const (
    WifiEventClientConnected    WifiClientEvent = "wifi:client:connected"    // Client connected to AP
    WifiEventClientDisconnected WifiClientEvent = "wifi:client:disconnected" // Client disconnected from AP
)
```

---

## Usage Examples

### Tracking Connected Clients

```go
func Init(api sdkapi.IPluginApi) error {
    // Track connected clients in memory
    connectedClients := make(map[string]time.Time)
    var mu sync.Mutex
    
    api.Wifi().OnWifiClientEvent(sdkapi.WifiEventClientConnected, func(client sdkapi.IWifiClient) {
        mu.Lock()
        connectedClients[client.MacAddress()] = time.Now()
        mu.Unlock()
        
        api.Logger().Info("Client %s connected at %s", 
            client.MacAddress(), time.Now().Format(time.RFC3339))
    })
    
    api.Wifi().OnWifiClientEvent(sdkapi.WifiEventClientDisconnected, func(client sdkapi.IWifiClient) {
        mu.Lock()
        if connectedAt, ok := connectedClients[client.MacAddress()]; ok {
            duration := time.Since(connectedAt)
            api.Logger().Info("Client %s disconnected after %v", 
                client.MacAddress(), duration)
            delete(connectedClients, client.MacAddress())
        }
        mu.Unlock()
    })
    
    return nil
}
```

### Auto-Connect Returning Clients

```go
func Init(api sdkapi.IPluginApi) error {
    api.Wifi().OnWifiClientEvent(sdkapi.WifiEventClientConnected, func(client sdkapi.IWifiClient) {
        ctx := context.Background()
        mac := client.MacAddress()
        
        // Check if this device has an existing session
        device, err := api.SessionsMgr().FindDeviceByMac(ctx, mac)
        if err != nil {
            // New device, no action needed
            return
        }
        
        // Check for available sessions
        sessions, err := api.SessionsMgr().GetAvailableSessions(ctx, device.ID())
        if err != nil || len(sessions) == 0 {
            return
        }
        
        // Auto-connect the returning client
        err = api.SessionsMgr().Connect(ctx, device, "Welcome back!")
        if err != nil {
            api.Logger().Error("Failed to auto-connect device %s: %v", mac, err)
            return
        }
        
        api.Logger().Info("Auto-connected returning client: %s", mac)
    })
    
    return nil
}
```

### Sending Notifications on Disconnect

```go
func Init(api sdkapi.IPluginApi) error {
    api.Wifi().OnWifiClientEvent(sdkapi.WifiEventClientDisconnected, func(client sdkapi.IWifiClient) {
        ctx := context.Background()
        mac := client.MacAddress()
        
        // Find the device
        device, err := api.SessionsMgr().FindDeviceByMac(ctx, mac)
        if err != nil {
            return
        }
        
        // Check if device had an active session
        sessions, err := api.SessionsMgr().GetActiveSessions(ctx, device.ID())
        if err != nil || len(sessions) == 0 {
            return
        }
        
        // Log the unexpected disconnect
        api.Logger().Warn("Active client disconnected unexpectedly: %s", mac)
        
        // Optionally send admin notification
        api.Notification().NotifyAdmin(ctx, sdkapi.NotificationParams{
            Title:   "Client Disconnected",
            Message: fmt.Sprintf("Client %s disconnected with active session", mac),
            Type:    sdkapi.NotificationTypeWarning,
        })
    })
    
    return nil
}
```

---

## Technical Details

### Event Source

WiFi events are captured from the hostapd control interface. The system monitors all hostapd interfaces in `/var/run/hostapd/` and automatically:

- Discovers new WiFi interfaces as they become available
- Reconnects if hostapd restarts
- Handles multiple WiFi radios (2.4GHz, 5GHz, etc.)

### Event Format

Events are received when hostapd reports:

- `AP-STA-CONNECTED <mac_address>` - Client associated with access point
- `AP-STA-DISCONNECTED <mac_address>` - Client disassociated from access point

---

## Related

- [ISessionsMgrApi](./sessions-mgr-api.md) - Managing client sessions
- [IClientDevice](./client-device.md) - Device information
- [INetworkApi](./network-api.md) - Network operations
