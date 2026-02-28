# IFirewallAPI

The `IFirewallAPI` provides methods to manage firewall rules for client devices in the Flare Hotspot system. It allows you to control network access using Destination IP Groups, which provide efficient firewall management for services with multiple IPs.

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
- `hostname` (string) - The hostname to resolve (e.g., "api.example.com")

**Returns:**
- `[]string` - List of IP addresses
- `error` - Error if resolution fails

```go
ips, err := api.Firewall().ResolveHostnameToIps("api.example.com")
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

## Destination IP Groups

Destination IP Groups provide an efficient way to manage firewall access for multiple destination IPs as a single unit. Instead of creating individual rules for each IP, you create a group once and add/remove clients from the group.

### Data Types

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
- `ips` (...string) - Initial set of destination IP addresses (variadic)

**Returns:**
- `error` - Error if group already exists or creation fails

```go
// Create a group for an external API service
err := api.Firewall().CreateDstIpGroup("my-api-service", 
    "104.21.45.123", 
    "172.67.132.45",
    "2606:4700:3030::6815:2d7b",
)
if err != nil {
    // Handle error - group may already exist
}
```

**Notes:**
- Group names are slugified (e.g., "my-api-service" becomes "my_api_service" in nftables)
- Returns error if group with same name already exists
- Creates nftables sets and chains for the group atomically
- IPv4 and IPv6 addresses are automatically separated

### DstIpGroupExists

Checks if a named destination IP group exists. Useful for conditionally creating groups or verifying group availability before operations.

**Parameters:**
- `name` (string) - Name of the group to check

**Returns:**
- `bool` - True if group exists, false otherwise
- `error` - Error only if group name is invalid

```go
// Check if group exists before creating
exists, err := api.Firewall().DstIpGroupExists("my-api-service")
if err != nil {
    api.Logger().Error("Invalid group name: " + err.Error())
    return
}

if !exists {
    // Create the group
    err := api.Firewall().CreateDstIpGroup("my-api-service", "104.21.45.123")
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
- `ips` (...string) - IP addresses to add (variadic)

**Returns:**
- `error` - Error if group doesn't exist or operation fails

```go
// Add more IPs to existing group (e.g., after DNS re-resolution)
err := api.Firewall().AddIpsToDstIpGroup("my-api-service", "104.21.45.124")
if err != nil {
    api.Logger().Error("Failed to add IPs: " + err.Error())
}
```

**Notes:**
- Does not remove existing IPs - only adds new ones
- Duplicate IPs are automatically filtered using in-memory tracking
- IPs older than 12 hours are automatically flushed when new IPs are added
- Use `ChangeDstIpGroup` if you want to immediately replace all IPs

### ChangeDstIpGroup

Replaces all IP addresses in an existing destination IP group with a new set. All existing IPs are removed first.

**Parameters:**
- `name` (string) - Name of the existing group
- `ips` (...string) - New set of IP addresses (replaces all existing)

**Returns:**
- `error` - Error if group doesn't exist or operation fails

```go
// Replace all IPs in group (e.g., after DNS refresh)
newIPs, _ := api.Firewall().ResolveHostnameToIps("api.example.com")

err := api.Firewall().ChangeDstIpGroup("my-api-service", newIPs...)
if err != nil {
    api.Logger().Error("Failed to update IPs: " + err.Error())
}
```

**Notes:**
- Atomic operation - flushes existing IPs and adds new ones in single batch
- Safe to call even if IPs haven't changed
- Clients already in the group retain access to new IPs automatically

### AllowClientToDstIpGroup

Allows a specific client device to access all IPs in a named destination IP group. The client's MAC and IP are added to the group's client sets.

**Parameters:**
- `clnt` (DstIpGroupClient) - Client device information
- `groupName` (string) - Name of the destination IP group
- `timeoutSecs` (int) - Timeout in seconds (0 = permanent, >0 = auto-remove)

**Returns:**
- `error` - Error if group doesn't exist or operation fails

```go
clnt, _ := api.Http().GetClientDevice(r)

// Allow client to access all IPs in the my-api-service group for 5 minutes
err := api.Firewall().AllowClientToDstIpGroup(
    sdkapi.DstIpGroupClient{
        MacAddr: clnt.MacAddr(),
        IpAddr:  clnt.IpAddr(),
    },
    "my-api-service",
    300, // 5 minutes
)
if err != nil {
    api.Logger().Error("Failed to allow client: " + err.Error())
}
```

**Notes:**
- If client is already in the group, the timeout is reset
- Client automatically has access to all current and future IPs in the group
- Single operation grants access to all IPs in the group

### RemoveClientFromDstIpGroup

Removes access for a specific client device from a named destination IP group. The client's MAC and IP are removed from the group's client sets.

**Parameters:**
- `clnt` (DstIpGroupClient) - Client device information
- `groupName` (string) - Name of the destination IP group

**Returns:**
- `error` - Error if group doesn't exist or operation fails

```go
clnt, _ := api.Http().GetClientDevice(r)

// Remove client from my-api-service group
err := api.Firewall().RemoveClientFromDstIpGroup(
    sdkapi.DstIpGroupClient{
        MacAddr: clnt.MacAddr(),
        IpAddr:  clnt.IpAddr(),
    },
    "my-api-service",
)
if err != nil {
    api.Logger().Error("Failed to remove client: " + err.Error())
}
```

**Notes:**
- Cancels any active timeout timer for this client
- Safe to call even if client is not in the group (no error)
- Client immediately loses access to all IPs in the group

## Common Use Cases

### Portal Redirect with Temporary Firewall Access

When redirecting users to an external portal for registration, you need to temporarily open firewall access:

```go
const portalGroupName = "portal-service"

// At plugin init - create the IP group
func Init(api sdkapi.IPluginApi) error {
    ips, err := api.Firewall().ResolveHostnameToIps("portal.example.com")
    if err != nil {
        return err
    }
    
    return api.Firewall().CreateDstIpGroup(portalGroupName, ips...)
}

// Portal redirect handler
func PortalRedirectHandler(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        clnt, _ := api.Http().GetClientDevice(r)
        
        // Allow client to access portal for 5 minutes
        err := api.Firewall().AllowClientToDstIpGroup(
            sdkapi.DstIpGroupClient{
                MacAddr: clnt.MacAddr(),
                IpAddr:  clnt.IpAddr(),
            },
            portalGroupName,
            300, // 5 minutes
        )
        if err != nil {
            api.Logger().Error("Failed to open firewall: " + err.Error())
        }
        
        // Redirect to external portal
        http.Redirect(w, r, "https://portal.example.com/register", http.StatusFound)
    }
}
```

### Complete Example: External Service Plugin Pattern

This example shows the recommended pattern for plugins that need to open firewall access to an external service:

```go
package myplugin

import (
    sdkapi "sdk/api"
)

const externalServiceGroup = "my-external-service"

// Init creates the IP group at plugin startup
func Init(api sdkapi.IPluginApi) error {
    // Resolve the external service domain
    ips, err := api.Firewall().ResolveHostnameToIps("api.myservice.com")
    if err != nil {
        return err
    }

    // Create the destination IP group
    if err := api.Firewall().CreateDstIpGroup(externalServiceGroup, ips...); err != nil {
        api.Logger().Error("Failed to create IP group: " + err.Error())
        // Continue anyway - may already exist from previous run
    }

    // Start background DNS refresh (optional but recommended)
    go refreshIPsInBackground(api)

    return nil
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
        api.Firewall().ChangeDstIpGroup(externalServiceGroup, ips...)
    }
}

// PortalMiddleware opens firewall for client to access external service
func PortalMiddleware(api sdkapi.IPluginApi) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            clnt, _ := api.Http().GetClientDevice(r)

            // Single call to allow client to access all service IPs
            err := api.Firewall().AllowClientToDstIpGroup(
                sdkapi.DstIpGroupClient{
                    MacAddr: clnt.MacAddr(),
                    IpAddr:  clnt.IpAddr(),
                },
                externalServiceGroup,
                300, // 5 minute timeout
            )
            if err != nil {
                api.Logger().Error("Failed to open firewall: " + err.Error())
            }

            next.ServeHTTP(w, r)
        })
    }
}

// CallbackHandler closes firewall after external interaction completes
func CallbackHandler(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        clnt, _ := api.Http().GetClientDevice(r)

        // Single call to remove client from all service IPs
        api.Firewall().RemoveClientFromDstIpGroup(
            sdkapi.DstIpGroupClient{
                MacAddr: clnt.MacAddr(),
                IpAddr:  clnt.IpAddr(),
            },
            externalServiceGroup,
        )

        // Handle callback logic...
    }
}
```

## Technical Details

### NFTables Structure

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

### Performance Considerations

- **DNS Resolution:** Uses 10-second timeout with fallback from Cloudflare to Google DNS
- **Concurrent Access:** Thread-safe timer management with mutex locks
- **Efficient Operations:** Single nft command to add/remove clients regardless of IP count
- **Automatic Cleanup:** Clients automatically removed after timeout expiration

### Firewall Implementation

- **Backend:** Uses nftables for firewall management
- **Table:** Creates rules in the `inet internet` table
- **Traffic:** Bidirectional (client → destination and destination → client)
- **Ports:** All ports are opened (no port restrictions)
- **IP Support:** Both IPv4 and IPv6 are supported

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

// Create or update IP group
exists, _ := api.Firewall().DstIpGroupExists("my-service")
if !exists {
    if err := api.Firewall().CreateDstIpGroup("my-service", ips...); err != nil {
        api.Logger().Error("Failed to create group: " + err.Error())
        return
    }
} else {
    if err := api.Firewall().ChangeDstIpGroup("my-service", ips...); err != nil {
        api.Logger().Error("Failed to update group: " + err.Error())
        // Continue anyway - existing IPs still work
    }
}

// Allow client access
err = api.Firewall().AllowClientToDstIpGroup(
    sdkapi.DstIpGroupClient{
        MacAddr: clnt.MacAddr(),
        IpAddr:  clnt.IpAddr(),
    },
    "my-service",
    300,
)
if err != nil {
    api.Logger().Error("Failed to allow client: " + err.Error())
}
```

## Best Practices

1. **Always use timeouts for temporary access** - Set `timeoutSecs > 0` to prevent orphaned access
2. **Resolve hostnames dynamically** - Use `ResolveHostnameToIps()` instead of hardcoding IPs
3. **Handle DNS resolution failures** - Services may use CDN/multiple IPs
4. **Log firewall operations** - Use `api.Logger()` to track firewall changes
5. **Clean up on errors** - If registration fails, consider removing client from group
6. **Refresh IPs periodically** - Use `ChangeDstIpGroup` to update IPs from DNS
7. **Test with IPv4 and IPv6** - Both protocols are supported

## Related Interfaces

- [IClientDevice](./client-device.md) - Represents a client device
- [ILoggerApi](./logger-api.md) - For logging firewall operations
- [ISessionsMgrApi](./sessions-mgr-api.md) - For managing client sessions
