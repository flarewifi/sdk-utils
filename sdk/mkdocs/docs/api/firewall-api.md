# IFirewallAPI

The `IFirewallAPI` lets plugins control which client devices can reach the network, and what they can reach, on a machine running behind a captive portal. By default the portal blocks every unauthenticated device; this API is how a plugin opens, scopes, or denies that access.

It is backed by **nftables** (the `inet internet` table) and is built for the constraints of an OpenWRT router: rules use hash-set lookups instead of per-packet connection tracking, so opening access for thousands of clients stays cheap and low-latency. All operations are serialized internally, so the API is safe to call concurrently from multiple handlers.

It groups into three capabilities, smallest scope first:

- **Destination IP Groups** — allow a set of clients to reach a *specific set of destination IPs* (e.g. a payment provider or a portal host), and nothing else. Create the group once, then add/remove clients (optionally with an auto-expiry timeout). Best for scoped, pre-auth access to known services. See [Destination IP Groups](#destination-ip-groups).
- **Service Ports** — allow clients to reach a *protocol + port* (e.g. UDP/123 NTP, TCP+UDP/53 DNS), optionally restricted to given destination IPs. Best for the handful of services a device needs *before* it authenticates. See [Service Port Management](#service-port-management).
- **MAC Access Control** — grant or deny a device *full* internet by MAC: `AllowMAC`/`DisallowMAC` whitelist a device past the portal, while `BlockMAC`/`UnblockMAC` are an absolute deny that overrides even an active paid session. See [MAC Access Control](#mac-access-control).
- **Custom Chain Attachment Points** — when none of the above fit, get a stable, named place in the shared table to `jump` into your own fully custom chain (own sets, own DNAT, own terminal verdicts) without touching core's `forward`/`prerouting` chains directly. See [Custom Chain Attachment Points](#custom-chain-attachment-points).

Rules created through this API live in nftables only and are **wiped on reboot** (and on firewall reset) — the plugin owns persistence and must re-apply what it granted on startup (see the `OnReady` requirement immediately below).

!!! danger "Call every method here from `OnReady`, not `Init()`"
    Every method in this API — `CreateDstIpGroup`, `CreateServicePort`, `AllowMAC`/`BlockMAC`,
    and the custom chain attachment points — needs the shared `inet internet` table (and, for
    MAC access control, its `whitelist_macs`/`blocked_macs` sets) to already exist. That table
    is created by `nftables.Setup()`, which runs **after** every plugin's `Init()` during boot.
    Re-apply persisted grants from `api.Network().OnReady(func() { ... })` instead — the same
    pattern already used by the whitelist/tailscale/coinslot plugins.

    Every method here returns an error if called this early (the underlying set/chain
    doesn't exist yet), so a plugin that checks its return value will notice — but nothing
    else warns you at compile time, and it is easy to miss if the error is only logged
    (see the `AllowMAC`/`BlockMAC` notes below on what an early call does and does not do).

!!! danger "Do not manage WiFi sessions with this API"
    `IFirewallAPI` works at the packet level — it opens and closes raw network access. It does **not** create, track, time, account for, pause/resume, or expire client **sessions**. For anything session-related use [`ISessionsMgrApi`](./sessions-mgr-api.md) (via `api.SessionsMgr()`), which owns the session lifecycle (time/data limits, vouchers, pause/resume, expiry) **and drives the firewall for you**. Reaching for `AllowMAC`/`BlockMAC` to grant timed internet bypasses session accounting and desyncs the portal — use `IFirewallAPI` only for access *outside* a session (scoped service access, pre-auth ports, whitelist bypass, hard blocks).

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

Represents a client device for group operations. Supports dual-stack (IPv4 + IPv6) devices — populate whichever address fields are present and both will be registered in the firewall.

```go
type DstIpGroupClient struct {
    MacAddr  string // Client device MAC address
    IpAddr   string // Primary IP – backward-compatible fallback (prefer Ipv4Addr/Ipv6Addr)
    Ipv4Addr string // IPv4 address for return traffic filtering (empty if IPv6-only)
    Ipv6Addr string // IPv6 address for return traffic filtering (empty if IPv4-only)
}
```

!!! note "Dual-Stack Usage"
    For dual-stack devices, always populate both `Ipv4Addr` and `Ipv6Addr` so that return traffic from the destination back to the client is correctly permitted for both protocols. Use `clnt.Ipv4Addr()` and `clnt.Ipv6Addr()` from `IClientDevice`.

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
- Safe to call again after a process restart (e.g. from `OnReady` on every boot) —
  automatically flushes and rebuilds this group's own kernel-side chains and sets
  first, so rules never duplicate across restarts
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

// Allow client to access all IPs in the my-api-service group for 5 minutes.
// Populate both Ipv4Addr and Ipv6Addr for dual-stack devices.
err := api.Firewall().AllowClientToDstIpGroup(
    sdkapi.DstIpGroupClient{
        MacAddr:  clnt.MacAddr(),
        Ipv4Addr: clnt.Ipv4Addr(), // empty string if device has no IPv4
        Ipv6Addr: clnt.Ipv6Addr(), // empty string if device has no IPv6
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
- Both IPv4 and IPv6 return-traffic rules are created when the respective address fields are non-empty

### RemoveClientFromDstIpGroup

Removes access for a specific client device from a named destination IP group. The client's MAC and IP are removed from the group's client sets.

**Parameters:**
- `clnt` (DstIpGroupClient) - Client device information
- `groupName` (string) - Name of the destination IP group

**Returns:**
- `error` - Error if group doesn't exist or operation fails

```go
clnt, _ := api.Http().GetClientDevice(r)

// Remove client from my-api-service group (both IPv4 and IPv6 entries are removed).
err := api.Firewall().RemoveClientFromDstIpGroup(
    sdkapi.DstIpGroupClient{
        MacAddr:  clnt.MacAddr(),
        Ipv4Addr: clnt.Ipv4Addr(),
        Ipv6Addr: clnt.Ipv6Addr(),
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

### DeleteDstIpGroup

Removes a named destination IP group and all its nftables infrastructure. All clients currently allowed access through this group will immediately lose access.

**Parameters:**
- `name` (string) - Name of the group to delete

**Returns:**
- `error` - Error if group doesn't exist or operation fails

```go
// Delete a destination IP group and cleanup all resources
err := api.Firewall().DeleteDstIpGroup("my-api-service")
if err != nil {
    api.Logger().Error("Failed to delete group: " + err.Error())
}
```

**Notes:**
- Cancels all scheduled automatic removals for clients in this group
- Removes all nftables sets, chains, and jump rules for the group
- All clients in the group immediately lose access
- Cannot be undone - group must be recreated with `CreateDstIpGroup`
- Safe to call even if no clients are in the group

## Service Port Management

Service Port methods allow you to create named service definitions (protocol + port) and then grant/revoke client access to those services. This follows the same pattern as Destination IP Groups, providing efficient firewall management for services. This is particularly useful for allowing pre-authentication access to essential services like NTP (time sync) and DNS.

### CreateServicePort

Creates a named service port definition with dedicated nftables infrastructure. The service port must be created before clients can be granted access to it.

**Parameters:**
- `name` (string) - Unique name for the service port (will be slugified for nftables compatibility)
- `protocols` ([]string) - Array of protocols: "tcp", "udp", or both (at least one required)
- `port` (int) - Destination port number (1-65535)
- `dstIPs` (...string) - Optional destination IPs to restrict access (empty = any destination)

**Returns:**
- `error` - Error if service port already exists or validation fails

```go
// Create an NTP service port (UDP/123, any destination)
err := api.Firewall().CreateServicePort("ntp", []string{"udp"}, 123)
if err != nil {
    api.Logger().Error("Failed to create NTP service port: " + err.Error())
}

// Create a DNS service port with specific servers (TCP+UDP/53)
err = api.Firewall().CreateServicePort(
    "dns",
    []string{"tcp", "udp"},
    53,
    "8.8.8.8", "8.8.4.4", "1.1.1.1", "1.0.0.1",
)
if err != nil {
    api.Logger().Error("Failed to create DNS service port: " + err.Error())
}
```

**Notes:**
- Service port names are slugified (e.g., "my-service" becomes "my_service" in nftables)
- Returns error if service port with same name already exists
- Safe to call again after a process restart (e.g. from `OnReady` on every boot) —
  automatically flushes and rebuilds this service port's own kernel-side chains and
  sets first, so rules never duplicate across restarts
- Creates nftables sets and chains atomically
- IPv4 and IPv6 addresses are automatically separated
- Once created, any number of clients can be added/removed efficiently

### ServicePortExists

Checks if a named service port exists. Useful for conditionally creating service ports or verifying availability before operations.

**Parameters:**
- `name` (string) - Name of the service port to check

**Returns:**
- `bool` - True if service port exists, false otherwise
- `error` - Error only if service port name is invalid

```go
// Check if service port exists before creating
exists, err := api.Firewall().ServicePortExists("ntp")
if err != nil {
    api.Logger().Error("Invalid service port name: " + err.Error())
    return
}

if !exists {
    // Create the service port
    err := api.Firewall().CreateServicePort("ntp", []string{"udp"}, 123)
    if err != nil {
        api.Logger().Error("Failed to create service port: " + err.Error())
    }
}
```

**Notes:**
- Only validates the service port name - does not query nftables
- Uses in-memory tracking, so very fast (no shell commands)
- Returns false for service ports that were never created

### DeleteServicePort

Removes a named service port and all its nftables infrastructure. All clients currently allowed access through this service port will immediately lose access.

**Parameters:**
- `name` (string) - Name of the service port to delete

**Returns:**
- `error` - Error if service port doesn't exist or operation fails

```go
// Delete a service port and cleanup all resources
err := api.Firewall().DeleteServicePort("ntp")
if err != nil {
    api.Logger().Error("Failed to delete service port: " + err.Error())
}
```

**Notes:**
- Cancels all scheduled automatic removals for clients using this service port
- Removes all nftables sets, chains, and jump rules for the service port
- All clients immediately lose access to this service
- Cannot be undone - service port must be recreated with `CreateServicePort`
- Safe to call even if no clients are using the service port

### AllowClientToServicePort

Allows a specific client device to access a named service port. The client's MAC and IP are added to the service port's client sets.

**Parameters:**
- `clnt` (DstIpGroupClient) - Client device information
- `servicePortName` (string) - Name of the service port
- `timeoutSecs` (int) - Timeout in seconds (0 = permanent, >0 = auto-remove)

**Returns:**
- `error` - Error if service port doesn't exist or operation fails

```go
clnt, _ := api.Http().GetClientDevice(r)

// Allow client to access NTP service for 60 seconds
err := api.Firewall().AllowClientToServicePort(
    sdkapi.DstIpGroupClient{
        MacAddr:  clnt.MacAddr(),
        Ipv4Addr: clnt.Ipv4Addr(),
        Ipv6Addr: clnt.Ipv6Addr(),
    },
    "ntp",
    60, // 1 minute
)
if err != nil {
    api.Logger().Error("Failed to allow NTP access: " + err.Error())
}
```

**Notes:**
- Service port must exist before calling this method
- If client is already allowed, the timeout is reset (idempotent)
- Client automatically has access to all destination IPs defined in the service port
- Both IPv4 and IPv6 return-traffic rules are created when the respective address fields are non-empty

### RemoveClientFromServicePort

Removes access for a specific client device from a named service port. The client's MAC and IP are removed from the service port's client sets.

**Parameters:**
- `clnt` (DstIpGroupClient) - Client device information
- `servicePortName` (string) - Name of the service port

**Returns:**
- `error` - Error if service port doesn't exist or operation fails

```go
clnt, _ := api.Http().GetClientDevice(r)

// Remove client from NTP service
err := api.Firewall().RemoveClientFromServicePort(
    sdkapi.DstIpGroupClient{
        MacAddr:  clnt.MacAddr(),
        Ipv4Addr: clnt.Ipv4Addr(),
        Ipv6Addr: clnt.Ipv6Addr(),
    },
    "ntp",
)
if err != nil {
    api.Logger().Error("Failed to remove NTP access: " + err.Error())
}
```

**Notes:**
- Cancels any active timeout timer for this client
- Safe to call even if client is not using the service port (no error)
- Client immediately loses access to the service

### Common Service Port Definitions

Here are some commonly used service port definitions:

```go
// At plugin initialization - create service ports

// NTP - Network Time Protocol (for clock sync)
api.Firewall().CreateServicePort("ntp", []string{"udp"}, 123)

// DNS - Domain Name System (for name resolution, any server)
api.Firewall().CreateServicePort("dns", []string{"tcp", "udp"}, 53)

// DNS to specific servers (Google and Cloudflare)
api.Firewall().CreateServicePort(
    "dns-restricted",
    []string{"tcp", "udp"},
    53,
    "8.8.8.8", "8.8.4.4", "1.1.1.1", "1.0.0.1",
)

// DHCP - Dynamic Host Configuration Protocol
api.Firewall().CreateServicePort("dhcp", []string{"udp"}, 67)
```

### Pre-Authentication Service Access Example

This example shows how to set up and allow essential services for clients before they authenticate:

```go
// At plugin Init() - create service ports once
func Init(api sdkapi.IPluginApi) error {
    // Create NTP service port (required for HTTPS certificate validation)
    if err := api.Firewall().CreateServicePort("ntp", []string{"udp"}, 123); err != nil {
        return fmt.Errorf("failed to create NTP service port: %w", err)
    }
    
    // Create DNS service port with specific servers
    if err := api.Firewall().CreateServicePort(
        "dns-preauth",
        []string{"tcp", "udp"},
        53,
        "8.8.8.8", "1.1.1.1",
    ); err != nil {
        return fmt.Errorf("failed to create DNS service port: %w", err)
    }
    
    return nil
}

// Middleware to grant pre-auth service access
func PreAuthMiddleware(api sdkapi.IPluginApi) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            clnt, _ := api.Http().GetClientDevice(r)
            
            client := sdkapi.DstIpGroupClient{
                MacAddr:  clnt.MacAddr(),
                Ipv4Addr: clnt.Ipv4Addr(),
                Ipv6Addr: clnt.Ipv6Addr(),
            }
            
            // Grant access to pre-auth services with 5 minute timeout
            if err := api.Firewall().AllowClientToServicePort(client, "ntp", 300); err != nil {
                api.Logger().Error("Failed to allow NTP access: " + err.Error())
            }
            
            if err := api.Firewall().AllowClientToServicePort(client, "dns-preauth", 300); err != nil {
                api.Logger().Error("Failed to allow DNS access: " + err.Error())
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

## MAC Access Control

These methods control a device's internet access by MAC address. They form two
independent pairs:

- **Whitelist (allow / revoke):** `AllowMAC` ↔ `DisallowMAC` — bypass the captive portal so a device gets internet without a session.
- **Hard block (deny / undo):** `BlockMAC` ↔ `UnblockMAC` — an absolute deny that overrides everything, including an active session or a whitelist entry.

The pairs are independent: `DisallowMAC` only undoes an `AllowMAC` grant (a device that still has an active session keeps internet), while `BlockMAC` drops traffic *before* the session and whitelist accepts are evaluated, so it wins regardless of state. **Lift a block with `UnblockMAC`, not `DisallowMAC`** (and vice-versa). All four are ephemeral — they exist only until reboot, so persist state and re-apply it from `OnReady` on startup (see the danger box at the top of this page — calling `AllowMAC`/`BlockMAC` from `Init()` returns an error, since the sets they mutate don't exist until `nftables.Setup()` has run).

### AllowMAC

Opens the firewall for a MAC address, bypassing the captive portal. Grants working **bidirectional** internet on its own: it resolves the device's current IP from the machine's DHCP/ARP/NDP tables, registers it for return traffic, and keeps it in sync as the IP changes (via the connect hook and a periodic reconcile). The caller does not manage IP sets.

**Parameters:**
- `mac` (string) - MAC address to allow (any common format, will be normalized)

**Returns:**
- `error` - Error if MAC format is invalid, or if the whitelist could not be updated in
  nftables (including if called before `nftables.Setup()` has run — see the `OnReady`
  danger box at the top of this page)

```go
// Allow a device to bypass the captive portal
err := api.Firewall().AllowMAC("aa:bb:cc:dd:ee:ff")
if err != nil {
    api.Logger().Error("Failed to allow MAC: " + err.Error())
}
```

**Notes:**
- Idempotent — calling twice for the same MAC is safe
- MAC is validated and normalized to uppercase before applying
- Does NOT persist across reboots — caller must re-apply on startup
- Handles both upload and return traffic internally — no separate IP management needed
- Independent of sessions: access persists until `DisallowMAC` (or until `BlockMAC` overrides it)

### DisallowMAC

Revokes an `AllowMAC` grant: removes the MAC from the whitelist bypass and clears the return-traffic IPs tracked for it. It is **not** a block — if the device still has an active session it keeps internet through the session path.

**Parameters:**
- `mac` (string) - MAC address to revoke (any common format, will be normalized)

**Returns:**
- `error` - Error if MAC format is invalid

```go
// Revoke a previously granted whitelist bypass
err := api.Firewall().DisallowMAC("aa:bb:cc:dd:ee:ff")
if err != nil {
    api.Logger().Error("Failed to disallow MAC: " + err.Error())
}
```

**Notes:**
- Idempotent — calling for a non-whitelisted MAC is safe (no error)
- Clears the return-traffic IPs core tracked for the MAC
- Does NOT block the device — use `BlockMAC` for an absolute deny

### BlockMAC

Absolutely denies internet access to a MAC, **regardless of whether the device has an active session or is whitelisted**. The deny is registered in `block_forward` — the chain wired into `plugin_forward_before`, the very first attachment point in `forward` — so it is evaluated above every accept rule and always wins. In-flight connections are cut immediately (conntrack is flushed). Reverse with `UnblockMAC`.

**Parameters:**
- `mac` (string) - MAC address to block (any common format, will be normalized)

**Returns:**
- `error` - Error if MAC format is invalid, or if the block could not be applied in
  nftables (including if called before `nftables.Setup()` has run — see the `OnReady`
  danger box at the top of this page)

```go
// Hard-block a device even if it has a paid session or is whitelisted
err := api.Firewall().BlockMAC("aa:bb:cc:dd:ee:ff")
if err != nil {
    api.Logger().Error("Failed to block MAC: " + err.Error())
}
```

**Notes:**
- Overrides BOTH the session accept and the whitelist accept — an absolute deny
- The upload deny is keyed on the MAC, so the block survives the device changing IP
- Cuts existing connections immediately (flushes conntrack for the device's IPs)
- Idempotent; MAC is validated and normalized to uppercase
- Does NOT persist across reboots — re-apply on startup if the block must survive a restart

### UnblockMAC

Removes a `BlockMAC` hard block, restoring whatever access the device would otherwise have (an active session and/or a whitelist entry). It grants no access on its own.

**Parameters:**
- `mac` (string) - MAC address to unblock (any common format, will be normalized)

**Returns:**
- `error` - Error if MAC format is invalid

```go
// Lift a hard block
err := api.Firewall().UnblockMAC("aa:bb:cc:dd:ee:ff")
if err != nil {
    api.Logger().Error("Failed to unblock MAC: " + err.Error())
}
```

**Notes:**
- Idempotent — calling for a non-blocked MAC is safe (no error)
- Removes the upload deny and the download IP denies added at block time
- Grants nothing by itself — the device regains access only via its session/whitelist

## Custom Chain Attachment Points

Destination IP groups, service ports, and MAC access control cover the common cases.
When a plugin needs a **fully custom** nftables rule — its own sets, its own DNAT, its
own terminal accept/drop logic — that a fixed group/port/MAC primitive can't express,
these four methods give it a stable, named attachment point in the shared `inet
internet` table without letting it touch core's own `forward`/`prerouting` chains
directly.

Each method creates a regular chain (if it doesn't already exist) owned entirely by the
caller, and wires a `jump` into one of four **generic** chains core itself creates and
positions once in `Setup()`: `plugin_prerouting_before`, `plugin_prerouting_after`,
`plugin_forward_before`, `plugin_forward_after`. The caller is responsible for every
rule inside its own chain (added via its own direct `nft` calls) — these methods only
create the chain and register the jump into the right generic chain.

!!! danger "Must be called from `OnReady`, not `Init()`"
    The shared table — and these four generic chains — don't exist until
    `nftables.Setup()` has completed, which happens after `Init()` normally runs. Call
    these methods from `api.Network().OnReady(func() { ... })` instead, the same pattern
    already used by the whitelist/tailscale/coinslot plugins.

**Where each attachment point lands relative to core's own rules:**

| Method | Generic chain | Position |
|--------|--------------|----------|
| `AddForwardChainBeforeInternet` | `plugin_forward_before` | Very top of `forward` — before even the hard-block (`BlockMAC`) rules. A terminal verdict here short-circuits core's forward logic entirely for matching packets. |
| `AddForwardChainAfterInternet` | `plugin_forward_after` | End of `forward` — after hard-block/whitelist/session verdict-map rules, but before the chain's own `drop` policy. |
| `AddPreRoutingChainBeforeInternet` | `plugin_prerouting_before` | Very top of `prerouting` — before the whitelist/session bypass and the captive-portal DNAT. |
| `AddPreRoutingChainAfterInternet` | `plugin_prerouting_after` | End of the rules `Setup()` builds for `prerouting`. **Note:** a WiFi captive-portal DNAT rule added later via `SetCaptivePortalTarget` lands *after* this jump too — "after" on prerouting is not the last word if a captive DNAT is also active. Use "Before" if your rule must run ahead of the captive redirect. |

A non-terminal jump (your chain's rules don't match) falls through and core's own rules
still apply as normal — nftables `jump` semantics only consume the packet if your chain
reaches a terminal verdict (`accept`/`drop`).

### AddForwardChainBeforeInternet

Creates (if needed) a chain named `chainName` in the shared `inet internet` table and
registers a jump to it from `plugin_forward_before`, which runs before every built-in
`forward` rule — including the hard-block checks.

**Parameters:**
- `chainName` (string) - Name of the plugin-owned chain to create and jump to

**Returns:**
- `error` - Error if the chain/jump could not be created

```go
func Init(api sdkapi.IPluginApi) error {
    api.Network().OnReady(func() {
        if err := api.Firewall().AddForwardChainBeforeInternet("pppoe_forward"); err != nil {
            api.Logger().Error("firewall: wire forward chain: " + err.Error())
        }
    })
    return nil
}
```

**Notes:**
- Idempotent — safe to call repeatedly with the same `chainName`
- The caller owns every rule inside `chainName`; this method does not add any rules to it
- Use this over `AddForwardChainAfterInternet` when core's own `forward` rules would
  otherwise accept/pass the traffic before your chain ever runs (e.g. an existing
  transparency-passthrough rule for the same interface)

### AddForwardChainAfterInternet

Same as `AddForwardChainBeforeInternet`, but registers into `plugin_forward_after`,
which runs after all of core's built-in `forward` rules (hard-block, session,
whitelist) and just before the chain's default `drop` policy.

**Parameters:**
- `chainName` (string) - Name of the plugin-owned chain to create and jump to

**Returns:**
- `error` - Error if the chain/jump could not be created

```go
api.Firewall().AddForwardChainAfterInternet("my_fallback_forward")
```

**Notes:**
- Idempotent — safe to call repeatedly with the same `chainName`
- Only reached for traffic core's own rules didn't already accept/drop
- Good fit for a "default deny with a few extra exceptions" pattern layered on top of
  core's existing session/whitelist logic

### AddPreRoutingChainBeforeInternet

Creates (if needed) a chain named `chainName` and registers a jump to it from
`plugin_prerouting_before`, which `Setup()` positions as the absolute topmost rule in
`prerouting` — before the whitelist/session bypass and before the captive-portal DNAT.

**Parameters:**
- `chainName` (string) - Name of the plugin-owned chain to create and jump to

**Returns:**
- `error` - Error if the chain/jump could not be created

```go
api.Firewall().AddPreRoutingChainBeforeInternet("pppoe_prerouting")
```

**Notes:**
- Idempotent — safe to call repeatedly with the same `chainName`
- Use this when your chain needs to intercept traffic (e.g. its own DNAT) before core's
  captive-portal redirect can apply

### AddPreRoutingChainAfterInternet

Same as `AddPreRoutingChainBeforeInternet`, but registers into
`plugin_prerouting_after`, which `Setup()` appends after its own built-in `prerouting`
rules.

**Parameters:**
- `chainName` (string) - Name of the plugin-owned chain to create and jump to

**Returns:**
- `error` - Error if the chain/jump could not be created

```go
api.Firewall().AddPreRoutingChainAfterInternet("my_late_prerouting")
```

**Notes:**
- Idempotent — safe to call repeatedly with the same `chainName`
- A captive-portal DNAT rule added later via `SetCaptivePortalTarget` (WiFi hotspot)
  still lands after this jump — don't rely on "after" meaning "last" if that DNAT is
  active

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
        
        // Allow client to access portal for 5 minutes (dual-stack: both IPv4 and IPv6)
        err := api.Firewall().AllowClientToDstIpGroup(
            sdkapi.DstIpGroupClient{
                MacAddr:  clnt.MacAddr(),
                Ipv4Addr: clnt.Ipv4Addr(),
                Ipv6Addr: clnt.Ipv6Addr(),
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

            // Single call to allow client to access all service IPs (dual-stack)
            err := api.Firewall().AllowClientToDstIpGroup(
                sdkapi.DstIpGroupClient{
                    MacAddr:  clnt.MacAddr(),
                    Ipv4Addr: clnt.Ipv4Addr(),
                    Ipv6Addr: clnt.Ipv6Addr(),
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

        // Single call to remove client from all service IPs (dual-stack)
        api.Firewall().RemoveClientFromDstIpGroup(
            sdkapi.DstIpGroupClient{
                MacAddr:  clnt.MacAddr(),
                Ipv4Addr: clnt.Ipv4Addr(),
                Ipv6Addr: clnt.Ipv6Addr(),
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

### MAC Access Control Precedence

The `forward` chain evaluates rules top-down, first terminal verdict wins. MAC
access-control rules sit in this order, which is why a hard block always beats a
grant:

1. **Hard block (drop)** — `blocked_macs` (by source MAC) and `blocked_client_ips_v4/v6` (by destination IP). `BlockMAC` fills these.
2. **Session accept** — `connected_macs_map` / `connected_ips_map` (set by the session manager).
3. **Whitelist accept** — `whitelist_macs` / `whitelist_client_ips_v4/v6`. `AllowMAC` fills these.
4. **Chain policy `drop`** — anything unmatched is blocked.

Because the block rules are first, `BlockMAC` overrides both an active session and a whitelist grant; `AllowMAC` only takes effect when the device is not blocked.

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

// Allow client access (dual-stack: both IPv4 and IPv6)
err = api.Firewall().AllowClientToDstIpGroup(
    sdkapi.DstIpGroupClient{
        MacAddr:  clnt.MacAddr(),
        Ipv4Addr: clnt.Ipv4Addr(),
        Ipv6Addr: clnt.Ipv6Addr(),
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
- [ISessionsMgrApi](./sessions-mgr-api.md) - **The** API for client WiFi sessions (lifecycle, limits, vouchers, pause/resume) — use it instead of `IFirewallAPI` whenever a session is involved
