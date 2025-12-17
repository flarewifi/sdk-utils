# IFirewallAPI

The `IFirewallAPI` provides methods to manage firewall rules for client devices in the Flare Hotspot system. It allows you to control network access by opening or closing firewall access to specific destination IPs for individual client devices, as well as resolve hostnames to IP addresses.

To get an instance of `IFirewallAPI`:

```go
firewallAPI := api.Firewall()
fmt.Println(firewallAPI) // IFirewallAPI
```

## IFirewallAPI Methods

The following methods are available in `IFirewallAPI`:

### ResolveHostnameToIps

Resolves a hostname to a list of IP addresses using Cloudflare (1.1.1.1) and Google (8.8.8.8) DNS servers. This is useful when you need to open firewall access to a domain rather than a specific IP.

**Parameters:**
- `hostname` (string) - The hostname to resolve (e.g., "cloud-sync.flarewifi.com")

**Returns:**
- `[]string` - List of IP addresses
- `error` - Error if resolution fails

```go
ips, err := api.Firewall().ResolveHostnameToIps("cloud-sync.flarewifi.com")
if err != nil {
    // handle error - DNS resolution failed
}

for _, ip := range ips {
    fmt.Printf("Resolved IP: %s\n", ip)
}
// Output might be:
// Resolved IP: 104.21.45.123
// Resolved IP: 172.67.132.45
```

**Use Case:**
This is particularly useful when integrating with external services that may use multiple IPs or CDN networks. Instead of hardcoding IPs, you can resolve them dynamically.

### OpenIpForClientDevice

Opens bidirectional firewall access for a specific client device to a destination IP address. All ports are opened for both outgoing (client → destination) and return (destination → client) traffic.

**Important:** If `TimeoutSecs` is `0`, the firewall rule is **permanent** and will not be automatically removed. If `TimeoutSecs > 0`, the rule is automatically removed after the specified duration.

**Parameters:**
- `params` ([OpenIpForClientDeviceParams](#openipforclientdeviceparams)) - Configuration for the firewall rule

**Returns:**
- `error` - Error if firewall rule creation fails

#### OpenIpForClientDeviceParams

```go
type OpenIpForClientDeviceParams struct {
    DestinationIp string // Destination IP address to allow access to
    IpAddr        string // Client device IP address (for return traffic filtering)
    MacAddr       string // Client device MAC address (for source traffic filtering)
    TimeoutSecs   int    // Timeout in seconds (0 = permanent, >0 = auto-remove after timeout)
}
```

#### Example: Temporary Access (5 minutes)

```go
clnt, _ := api.Http().GetClientDevice(r)

// Open firewall for 5 minutes to allow portal registration
err := api.Firewall().OpenIpForClientDevice(sdkapi.OpenIpForClientDeviceParams{
    DestinationIp: "104.21.45.123",
    IpAddr:        clnt.IpAddr(),
    MacAddr:       clnt.MacAddr(),
    TimeoutSecs:   300, // 5 minutes - auto-removed after timeout
})
if err != nil {
    // handle error - firewall rule creation failed
}

// Client can now access 104.21.45.123 for 5 minutes
// Rule automatically removed after timeout
```

#### Example: Permanent Access

```go
clnt, _ := api.Http().GetClientDevice(r)

// Open permanent firewall access (no timeout)
err := api.Firewall().OpenIpForClientDevice(sdkapi.OpenIpForClientDeviceParams{
    DestinationIp: "8.8.8.8",
    IpAddr:        clnt.IpAddr(),
    MacAddr:       clnt.MacAddr(),
    TimeoutSecs:   0, // 0 = permanent rule, never auto-removed
})
if err != nil {
    // handle error
}

// Client can now access 8.8.8.8 permanently
// Rule persists until manually closed with CloseIpForClientDevice
```

#### Example: Opening Access to Multiple IPs

```go
clnt, _ := api.Http().GetClientDevice(r)

// Resolve domain to IPs
ips, err := api.Firewall().ResolveHostnameToIps("cloud-sync.flarewifi.com")
if err != nil {
    // handle error
}

// Open firewall for each resolved IP
for _, ip := range ips {
    err := api.Firewall().OpenIpForClientDevice(sdkapi.OpenIpForClientDeviceParams{
        DestinationIp: ip,
        IpAddr:        clnt.IpAddr(),
        MacAddr:       clnt.MacAddr(),
        TimeoutSecs:   300, // 5 minutes
    })
    if err != nil {
        logger.Error("Failed to open firewall for IP: " + ip)
    }
}
```

**Notes:**
- If a rule already exists, the timeout is reset (existing timer cancelled and new one started)
- Rules are bidirectional - both outgoing and return traffic are allowed
- IPv4 and IPv6 are both supported
- Uses nftables under the hood for firewall management

### CloseIpForClientDevice

Removes firewall access for a specific client device to a destination IP address. This manually closes a firewall rule that was previously opened.

**Parameters:**
- `params` ([CloseIpForClientDeviceParams](#closeipforclientdeviceparams)) - Configuration for closing the firewall rule

**Returns:**
- `error` - Error if firewall rule removal fails

#### CloseIpForClientDeviceParams

```go
type CloseIpForClientDeviceParams struct {
    DestinationIp string // Destination IP address to close access to
    MacAddr       string // Client device MAC address
}
```

#### Example: Manually Closing Firewall Access

```go
clnt, _ := api.Http().GetClientDevice(r)

// Close firewall access to specific IP
err := api.Firewall().CloseIpForClientDevice(sdkapi.CloseIpForClientDeviceParams{
    DestinationIp: "104.21.45.123",
    MacAddr:       clnt.MacAddr(),
})
if err != nil {
    // handle error - firewall rule removal failed
}

// Client can no longer access 104.21.45.123
```

**Notes:**
- If a timeout timer is active for this rule, it will be cancelled
- If the rule doesn't exist, the method returns successfully (no error)
- Both outgoing and return traffic rules are removed

## Common Use Cases

### Portal Redirect with Temporary Firewall Access

When redirecting users to an external portal for registration, you need to temporarily open firewall access:

```go
func PortalRedirectHandler(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        clnt, _ := api.Http().GetClientDevice(r)
        
        // Resolve portal domain to IPs
        ips, err := api.Firewall().ResolveHostnameToIps("portal.example.com")
        if err != nil {
            // handle error
            return
        }
        
        // Open firewall for 5 minutes to allow registration
        for _, ip := range ips {
            api.Firewall().OpenIpForClientDevice(sdkapi.OpenIpForClientDeviceParams{
                DestinationIp: ip,
                IpAddr:        clnt.IpAddr(),
                MacAddr:       clnt.MacAddr(),
                TimeoutSecs:   300, // 5 minutes
            })
        }
        
        // Redirect to external portal
        http.Redirect(w, r, "https://portal.example.com/register", http.StatusFound)
    }
}
```

### Whitelisting Specific Services

Allow clients to access specific services (DNS, NTP, etc.) even when disconnected:

```go
func AllowEssentialServices(api sdkapi.IPluginApi, clnt sdkapi.IClientDevice) {
    essentialIPs := []string{
        "8.8.8.8",       // Google DNS
        "1.1.1.1",       // Cloudflare DNS
        "time.google.com", // NTP server
    }
    
    for _, destination := range essentialIPs {
        // Check if it's a hostname or IP
        ips := []string{destination}
        if net.ParseIP(destination) == nil {
            // It's a hostname, resolve it
            resolvedIPs, err := api.Firewall().ResolveHostnameToIps(destination)
            if err == nil {
                ips = resolvedIPs
            }
        }
        
        // Open permanent access to essential services
        for _, ip := range ips {
            api.Firewall().OpenIpForClientDevice(sdkapi.OpenIpForClientDeviceParams{
                DestinationIp: ip,
                IpAddr:        clnt.IpAddr(),
                MacAddr:       clnt.MacAddr(),
                TimeoutSecs:   0, // Permanent
            })
        }
    }
}
```

### Time-Limited VIP Access

Grant premium users temporary access to specific IPs:

```go
func GrantVIPAccess(api sdkapi.IPluginApi, clnt sdkapi.IClientDevice, vipServerIP string, durationMinutes int) error {
    timeoutSecs := durationMinutes * 60
    
    return api.Firewall().OpenIpForClientDevice(sdkapi.OpenIpForClientDeviceParams{
        DestinationIp: vipServerIP,
        IpAddr:        clnt.IpAddr(),
        MacAddr:       clnt.MacAddr(),
        TimeoutSecs:   timeoutSecs,
    })
}

// Usage:
// Grant 30-minute VIP access
err := GrantVIPAccess(api, clnt, "203.0.113.100", 30)
```

## Technical Details

### Firewall Implementation

- **Backend:** Uses nftables for firewall management
- **Table:** Creates rules in the `inet internet` table
- **Chains:** Uses `open_ip_prerouting` and `open_ip_forward` chains
- **Traffic:** Bidirectional (client → destination and destination → client)
- **Ports:** All ports are opened (no port restrictions)

### Rule Lifecycle

1. **Rule Creation:**
   - Creates 4 nftables rules (2 for prerouting, 2 for forward)
   - Outgoing: client MAC → destination IP
   - Return: destination IP → client IP

2. **Timer Management:**
   - If `TimeoutSecs > 0`: Timer scheduled for automatic removal
   - If `TimeoutSecs = 0`: No timer, rule persists indefinitely
   - Timers stored in memory with mutex protection for thread safety

3. **Rule Removal:**
   - Automatic: Timer expires → calls `CloseIpForClientDevice`
   - Manual: Call `CloseIpForClientDevice` directly
   - Removes all 4 associated nftables rules

### Performance Considerations

- **DNS Resolution:** Uses 10-second timeout with fallback from Cloudflare to Google DNS
- **Concurrent Access:** Thread-safe timer management with mutex locks
- **Rule Reuse:** If rule exists, only timer is reset (no duplicate rules)
- **Cleanup:** Automatic cleanup on timer expiration

## Error Handling

```go
clnt, _ := api.Http().GetClientDevice(r)

// Example of comprehensive error handling
ips, err := api.Firewall().ResolveHostnameToIps("example.com")
if err != nil {
    api.Logger().Error("DNS resolution failed: " + err.Error())
    // Fallback to cached IPs or show error to user
    return
}

for _, ip := range ips {
    err := api.Firewall().OpenIpForClientDevice(sdkapi.OpenIpForClientDeviceParams{
        DestinationIp: ip,
        IpAddr:        clnt.IpAddr(),
        MacAddr:       clnt.MacAddr(),
        TimeoutSecs:   300,
    })
    
    if err != nil {
        api.Logger().Error("Failed to open firewall for " + ip + ": " + err.Error())
        // Continue with other IPs or handle error appropriately
    }
}
```

## Best Practices

1. **Always use timeouts for temporary access** - Set `TimeoutSecs > 0` to prevent orphaned rules
2. **Resolve hostnames dynamically** - Use `ResolveHostnameToIps()` instead of hardcoding IPs
3. **Handle DNS resolution failures** - Services may use CDN/multiple IPs
4. **Log firewall operations** - Use `api.Logger()` to track firewall changes
5. **Clean up on errors** - If registration fails, consider closing firewall access
6. **Use permanent rules sparingly** - Only for essential services (DNS, NTP)
7. **Test with IPv4 and IPv6** - Both protocols are supported

## Related Interfaces

- [IClientDevice](./client-device.md) - Represents a client device
- [ILoggerApi](./logger-api.md) - For logging firewall operations
- [ISessionsMgrApi](./sessions-mgr-api.md) - For managing client sessions
