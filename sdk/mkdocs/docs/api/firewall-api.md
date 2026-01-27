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
    DstIp string // Destination IP address to allow access to
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
    DstIp: "104.21.45.123",
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
    DstIp: "8.8.8.8",
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
        DstIp: ip,
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
    DstIp string // Destination IP address to close access to
    MacAddr       string // Client device MAC address
}
```

#### Example: Manually Closing Firewall Access

```go
clnt, _ := api.Http().GetClientDevice(r)

// Close firewall access to specific IP
err := api.Firewall().CloseIpForClientDevice(sdkapi.CloseIpForClientDeviceParams{
    DstIp: "104.21.45.123",
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
                DstIp: ip,
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
                DstIp: ip,
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
        DstIp: vipServerIP,
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

## Destination IP Groups

Destination IP Groups provide an efficient way to manage firewall access for multiple destination IPs as a single unit. Instead of creating individual rules for each IP, you create a group once and add/remove clients from the group.

### When to Use IP Groups vs Individual IPs

| Use Case | Approach |
|----------|----------|
| Single destination IP | `OpenIpForClientDevice` / `CloseIpForClientDevice` |
| Multiple IPs for same service (e.g., CDN, cloud service) | Destination IP Groups |
| IPs that change frequently (DNS-resolved domains) | Destination IP Groups with `ChangeDstIpGroup` |
| Many clients accessing same set of IPs | Destination IP Groups (more efficient) |

### Data Types

#### DstIpGroup

Represents a group of destination IP addresses, separated by IP version.

```go
type DstIpGroup struct {
    IPv4 []string // List of IPv4 addresses
    IPv6 []string // List of IPv6 addresses
}
```

#### DstIpGroupClient

Represents a client device for group operations.

```go
type DstIpGroupClient struct {
    MacAddr string // Client device MAC address
    IpAddr  string // Client device IP address (for return traffic filtering)
}
```

### CreateDstIpGroup

Creates a named group of destination IP addresses with dedicated nftables infrastructure. The group must be created before clients can be added to it.

**Parameters:**
- `name` (string) - Unique name for the group (will be slugified for nftables compatibility)
- `ips` (DstIpGroup) - Initial set of destination IP addresses

**Returns:**
- `error` - Error if group already exists or creation fails

```go
// Create a group for cloud-sync service
err := api.Firewall().CreateDstIpGroup("cloud-sync", sdkapi.DstIpGroup{
    IPv4: []string{"104.21.45.123", "172.67.132.45"},
    IPv6: []string{"2606:4700:3030::6815:2d7b"},
})
if err != nil {
    // Handle error - group may already exist
}
```

**Notes:**
- Group names are slugified (e.g., "cloud-sync" becomes "cloud_sync" in nftables)
- Returns error if group with same name already exists
- Creates nftables sets and chains for the group atomically

### DstIpGroupExists

Checks if a named destination IP group exists. Useful for conditionally creating groups or verifying group availability before operations.

**Parameters:**
- `name` (string) - Name of the group to check

**Returns:**
- `bool` - True if group exists, false otherwise
- `error` - Error only if group name is invalid

```go
// Check if group exists before creating
exists, err := api.Firewall().DstIpGroupExists("cloud-sync")
if err != nil {
    api.Logger().Error("Invalid group name: " + err.Error())
    return
}

if !exists {
    // Create the group
    err := api.Firewall().CreateDstIpGroup("cloud-sync", sdkapi.DstIpGroup{
        IPv4: []string{"104.21.45.123"},
    })
    if err != nil {
        api.Logger().Error("Failed to create group: " + err.Error())
    }
}
```

**Notes:**
- Only validates the group name - does not query nftables
- Uses in-memory tracking, so very fast (no shell commands)
- Returns false for groups that were never created through `CreateDstIpGroup`

### AddIpsToDstIpGroup

Adds IP addresses to an existing destination IP group. The new IPs are merged with existing IPs in the group.

**Parameters:**
- `name` (string) - Name of the existing group
- `ips` (DstIpGroup) - IP addresses to add

**Returns:**
- `error` - Error if group doesn't exist or operation fails

```go
// Add more IPs to existing group (e.g., after DNS re-resolution)
err := api.Firewall().AddIpsToDstIpGroup("cloud-sync", sdkapi.DstIpGroup{
    IPv4: []string{"104.21.45.124"}, // New IP discovered
})
if err != nil {
    api.Logger().Error("Failed to add IPs: " + err.Error())
}
```

**Notes:**
- Does not remove existing IPs - only adds new ones
- Duplicate IPs are handled gracefully by nftables (no error)
- Use `ChangeDstIpGroup` if you want to replace all IPs

### ChangeDstIpGroup

Replaces all IP addresses in an existing destination IP group with a new set. All existing IPs are removed first.

**Parameters:**
- `name` (string) - Name of the existing group
- `ips` (DstIpGroup) - New set of IP addresses (replaces all existing)

**Returns:**
- `error` - Error if group doesn't exist or operation fails

```go
// Replace all IPs in group (e.g., after DNS refresh)
newIPs, _ := api.Firewall().ResolveHostnameToIps("cloud-sync.example.com")
ipGroup := separateIPsByVersion(newIPs) // Helper to split IPv4/IPv6

err := api.Firewall().ChangeDstIpGroup("cloud-sync", ipGroup)
if err != nil {
    api.Logger().Error("Failed to update IPs: " + err.Error())
}
```

**Notes:**
- Atomic operation - flushes existing IPs and adds new ones in single batch
- Safe to call even if IPs haven't changed
- Clients already in the group retain access to new IPs automatically

### AllowClientDeviceToDstIpGroup

Allows a specific client device to access all IPs in a named destination IP group. The client's MAC and IP are added to the group's client sets.

**Parameters:**
- `clnt` (DstIpGroupClient) - Client device information
- `groupName` (string) - Name of the destination IP group
- `timeoutSecs` (int) - Timeout in seconds (0 = permanent, >0 = auto-remove)

**Returns:**
- `error` - Error if group doesn't exist or operation fails

```go
clnt, _ := api.Http().GetClientDevice(r)

// Allow client to access all IPs in the cloud-sync group for 5 minutes
err := api.Firewall().AllowClientDeviceToDstIpGroup(
    sdkapi.DstIpGroupClient{
        MacAddr: clnt.MacAddr(),
        IpAddr:  clnt.IpAddr(),
    },
    "cloud-sync",
    300, // 5 minutes
)
if err != nil {
    api.Logger().Error("Failed to allow client: " + err.Error())
}
```

**Notes:**
- If client is already in the group, the timeout is reset
- Client automatically has access to all current and future IPs in the group
- Much more efficient than calling `OpenIpForClientDevice` for each IP

### RemoveClientDeviceFromDstIpGroup

Removes access for a specific client device from a named destination IP group. The client's MAC and IP are removed from the group's client sets.

**Parameters:**
- `clnt` (DstIpGroupClient) - Client device information
- `groupName` (string) - Name of the destination IP group

**Returns:**
- `error` - Error if group doesn't exist or operation fails

```go
clnt, _ := api.Http().GetClientDevice(r)

// Remove client from cloud-sync group
err := api.Firewall().RemoveClientDeviceFromDstIpGroup(
    sdkapi.DstIpGroupClient{
        MacAddr: clnt.MacAddr(),
        IpAddr:  clnt.IpAddr(),
    },
    "cloud-sync",
)
if err != nil {
    api.Logger().Error("Failed to remove client: " + err.Error())
}
```

**Notes:**
- Cancels any active timeout timer for this client
- Safe to call even if client is not in the group (no error)
- Client immediately loses access to all IPs in the group

### Complete Example: Cloud Sync Plugin Pattern

This example shows the recommended pattern for plugins that need to open firewall access to a cloud service:

```go
package myplugin

import (
    sdkapi "sdk/api"
)

const cloudSyncGroupName = "my-cloud-service"

// Init creates the IP group at plugin startup
func Init(api sdkapi.IPluginApi) error {
    // Resolve the cloud service domain
    ips, err := api.Firewall().ResolveHostnameToIps("api.myservice.com")
    if err != nil {
        return err
    }

    // Separate IPv4 and IPv6
    ipGroup := separateIPsByVersion(ips)

    // Create the destination IP group
    if err := api.Firewall().CreateDstIpGroup(cloudSyncGroupName, ipGroup); err != nil {
        api.Logger().Error("Failed to create IP group: " + err.Error())
        // Continue anyway - may already exist from previous run
    }

    // Start background DNS refresh (optional but recommended)
    go refreshIPsInBackground(api)

    return nil
}

// separateIPsByVersion splits IPs into IPv4 and IPv6 groups
func separateIPsByVersion(ips []string) sdkapi.DstIpGroup {
    var result sdkapi.DstIpGroup
    for _, ip := range ips {
        if strings.Contains(ip, ":") {
            result.IPv6 = append(result.IPv6, ip)
        } else {
            result.IPv4 = append(result.IPv4, ip)
        }
    }
    return result
}

// refreshIPsInBackground periodically updates the IP group
func refreshIPsInBackground(api sdkapi.IPluginApi) {
    ticker := time.NewTicker(30 * time.Minute)
    for range ticker.C {
        ips, err := api.Firewall().ResolveHostnameToIps("api.myservice.com")
        if err != nil {
            api.Logger().Error("DNS refresh failed: " + err.Error())
            continue
        }
        ipGroup := separateIPsByVersion(ips)
        api.Firewall().ChangeDstIpGroup(cloudSyncGroupName, ipGroup)
    }
}

// PortalMiddleware opens firewall for client to access cloud service
func PortalMiddleware(api sdkapi.IPluginApi) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            clnt, _ := api.Http().GetClientDevice(r)

            // Single call to allow client to access all service IPs
            err := api.Firewall().AllowClientDeviceToDstIpGroup(
                sdkapi.DstIpGroupClient{
                    MacAddr: clnt.MacAddr(),
                    IpAddr:  clnt.IpAddr(),
                },
                cloudSyncGroupName,
                300, // 5 minute timeout
            )
            if err != nil {
                api.Logger().Error("Failed to open firewall: " + err.Error())
            }

            next.ServeHTTP(w, r)
        })
    }
}

// CallbackHandler closes firewall after cloud interaction completes
func CallbackHandler(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        clnt, _ := api.Http().GetClientDevice(r)

        // Single call to remove client from all service IPs
        api.Firewall().RemoveClientDeviceFromDstIpGroup(
            sdkapi.DstIpGroupClient{
                MacAddr: clnt.MacAddr(),
                IpAddr:  clnt.IpAddr(),
            },
            cloudSyncGroupName,
        )

        // Handle callback logic...
    }
}
```

### Technical Details: IP Groups

#### NFTables Structure

When you create a destination IP group named "my-service", the following nftables resources are created:

- **Sets:**
  - `dst_grp_my_service_v4` - IPv4 destination addresses
  - `dst_grp_my_service_v6` - IPv6 destination addresses
  - `dst_grp_my_service_macs` - Client MAC addresses
  - `dst_grp_my_service_client_ips_v4` - Client IPv4 addresses (return traffic)
  - `dst_grp_my_service_client_ips_v6` - Client IPv6 addresses (return traffic)

- **Chains:**
  - `dst_grp_my_service_prerouting` - Prerouting rules
  - `dst_grp_my_service_forward` - Forward rules

- **Jump Rules:**
  - From `prerouting` → `dst_grp_my_service_prerouting`
  - From `forward` → `dst_grp_my_service_forward`

#### Performance Benefits

| Operation | Individual IPs | IP Groups |
|-----------|---------------|-----------|
| Add client for N IPs | N nft commands | 1 nft command |
| Remove client for N IPs | N nft commands | 1 nft command |
| Update IPs (DNS refresh) | Complex tracking | Single `ChangeDstIpGroup` |
| Memory usage | N rules per client | 1 set entry per client |

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
        DstIp: ip,
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
