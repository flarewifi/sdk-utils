# Firewall Manipulation

The machine's firewall is a single [nftables](https://wiki.nftables.org/) table named
**`internet`** in the **`inet`** family (dual-stack — one rule set matches both IPv4 and
IPv6). Core creates and owns this table; plugins never write raw `nft` commands against
it directly. Instead, [`IFirewallAPI`](../api/firewall-api.md) (`api.Firewall()`) gives
you plugin-scoped primitives — **dst IP groups**, **service ports**, **MAC allow/block**,
and **custom chain attachment points** — that install their own chains and wire a single
jump rule into the core chains on your behalf.

This guide describes the chains core sets up so you know **where** your rules attach and
**in what order** they're evaluated. For the full method-by-method API reference, see
[IFirewallAPI](../api/firewall-api.md).

!!! warning "Never edit the core chains directly"
    `forward`, `prerouting`, and `postrouting` are managed by core and rebuilt on
    `Setup()`. A plugin should only ever **attach** to them via `api.Firewall()` — never
    flush, insert into, or delete rules from these chains directly. Doing so will be
    silently undone the next time core reconciles the table.

## Core chains

| Chain | Type / Hook | Priority | Policy | Purpose |
|-------|-------------|----------|--------|---------|
| `forward` | `filter` / `forward` | `-250` | **drop** | Main session gate. Everything routed *through* the machine (client ↔ internet) passes here. |
| `prerouting` | `nat` / `prerouting` | `-1` | accept | Bypass checks that run before any DNAT, plus the captive-portal HTTP redirect for unauthenticated clients. |
| `postrouting` | `filter` / `postrouting` | `0` | accept | Anti-tethering — forces TTL/hop-limit to `1` on egress through managed interfaces. |
| `plugin_forward_before` | regular (no hook) | — | — | Generic attachment point, `jump`ed to first in `forward`. Wired to a plugin's own chain via `AddForwardChainBeforeInternet`. Core's own hard-block logic (`BlockMAC`) is registered here first, ahead of any plugin. |
| `plugin_forward_after` | regular (no hook) | — | — | Generic attachment point, `jump`ed to last in `forward`. Wired via `AddForwardChainAfterInternet`. Core's own MAC allow-list (`AllowMAC`) is registered here first, ahead of any plugin. |
| `plugin_prerouting_before` | regular (no hook) | — | — | Generic attachment point, `jump`ed to first in `prerouting`. Wired via `AddPreRoutingChainBeforeInternet`. Core's own MAC allow-list (`AllowMAC`) is registered here first, ahead of any plugin. |
| `plugin_prerouting_after` | regular (no hook) | — | — | Generic attachment point, `jump`ed to last among `Setup()`'s own `prerouting` rules. Wired via `AddPreRoutingChainAfterInternet`. |

Core's own `AllowMAC`/`DisallowMAC`/`BlockMAC`/`UnblockMAC` are not special-cased against
`forward`/`prerouting` directly — they are implemented as reserved chains (holding the
allow/block set lookups) that register into `plugin_forward_before`/`after` and
`plugin_prerouting_before` through the exact same mechanism a plugin uses via
`AddForwardChainBeforeInternet` & co., just running first because core wires them during
boot, before any plugin's `OnReady` fires.

### `forward`

The traffic gate. Rules are evaluated in this order:

1. `jump plugin_forward_before` — **hard block** (MAC/IP in the block sets → `drop`) runs
   first here, beating everything including an active session or whitelist entry. Any
   plugin chain attached "before" runs after the hard block, in registration order.
2. Captive-portal bypass — a client on a captive interface reaching the portal's own IP.
3. Passthrough for traffic that isn't between two managed interfaces (e.g. an unmanaged
   or Tailscale-style interface).
4. **Session verdict maps** — a client with an active session (tracked by MAC for
   upload, by IP for download) → `accept`.
5. `jump plugin_forward_after` — the MAC allow-list (whitelisted MAC/IP → `accept`) runs
   first here; any plugin chain attached "after" runs after it.
6. Plugin-owned chains (dst IP groups, service ports) are appended here, after step 5.
7. Implicit policy: **drop**.

No `ct state` (conntrack) matching is used — session/whitelist/block state is tracked in
nftables sets and verdict maps for O(1) lookups, which is why `Connect`/`Disconnect`/
`AllowMAC`/`BlockMAC` all mutate sets rather than relying on connection tracking.

### `prerouting`

Runs before NAT. In order: `plugin_prerouting_before` (topmost — the MAC allow-list bypass
runs first here, then any plugin chain attached "before"), active-session bypass,
portal-IP bypass, `plugin_prerouting_after`, then the captive-portal redirect (port 80
only — port 443 is intentionally left alone, since intercepting TLS breaks the browser).
Plugin dst-IP-group chains are **inserted** at the top of this chain (before the captive
redirect), so an allowed client's traffic never gets redirected to the portal.

### `postrouting`

Unrelated to session/whitelist/block logic — a single anti-tethering rule that sets
`ip ttl 1` / `ip6 hoplimit 1` on packets leaving a managed interface, so a device tethering
off a connected client loses connectivity at the next hop.

## Sets backing the chains

These aren't things you manipulate directly, but knowing they exist helps explain why
rule order matters:

| Set | Holds | Used for |
|-----|-------|----------|
| `blocked_macs`, `blocked_client_ips_v4/v6` | Hard-blocked clients | `BlockMAC`/`UnblockMAC` |
| `connected_macs_map`, `connected_ips_map`, `connected_ips6_map` | Active sessions (verdict maps) | Session start/stop |
| `connected_macs_set` | Active sessions (membership only) | Fast `IsConnected` / prerouting bypass |
| `whitelist_macs`, `whitelist_client_ips_v4/v6` | Whitelisted clients | `AllowMAC`/`DisallowMAC` |
| `managed_ifaces`, `captive_ifaces` | Interfaces under enforcement | Interface configuration |
| `portal_ips_v4/v6` | The machine's own portal-serving IPs | Portal bypass rules |

## Adding custom rules on top of the core chains

Each `IFirewallAPI` capability creates **its own chain(s)** and attaches with a single
`jump` rule — it never touches the rules already in `forward`/`prerouting`. Where that
jump lands determines who your rule applies to:

- **Dst IP groups** — jump inserted into `prerouting` (before the portal redirect) and
  appended into `forward` (after the session/whitelist accepts). Use this to grant a
  client's device access to a specific set of destination IPs — including
  **unauthenticated** clients, since it runs before the drop policy regardless of session
  state.
- **Service ports** — jump appended into `forward` only. Use this to open a
  protocol/port combination (e.g. DNS, NTP) for clients that don't have a session yet.
- **MAC allow/block** — `AllowMAC`/`BlockMAC` add directly to the existing
  `whitelist_*`/`blocked_*` sets rather than creating a new chain per call, since they
  represent per-client overrides of the same bypass/block semantics core already
  evaluates. The chains backing these sets are themselves wired into
  `plugin_forward_before`/`after`/`plugin_prerouting_before` as the first registrant —
  see the table above — so the hard block always runs before any plugin's own "before"
  chain, and a plugin's own "before" chain always runs before the whitelist bypass (so
  it can still override a whitelisted client if it needs to).
- **Custom chain attachment points** — when a dst IP group, service port, or MAC
  allow/block doesn't fit (you need your own sets, your own DNAT, your own terminal
  verdict), `AddForwardChainBeforeInternet`/`AfterInternet` and
  `AddPreRoutingChainBeforeInternet`/`AfterInternet` create your own chain and jump into
  one of the four generic `plugin_*` chains listed in the table above — see
  [Custom Chain Attachment Points](../api/firewall-api.md#custom-chain-attachment-points)
  for the full reference.

### Example: pre-auth access to a fixed set of destinations (dst IP group)

Useful for a payment gateway or SSO provider a client must reach *before* they're
authenticated:

```go
const paymentGatewayGroup = "payment-gateway"

func setupFirewallGroup(api sdkapi.IPluginApi) error {
    ips, err := api.Firewall().ResolveHostnameToIps("checkout.example.com")
    if err != nil {
        return fmt.Errorf("resolve payment gateway host: %w", err)
    }
    return api.Firewall().CreateDstIpGroup(paymentGatewayGroup, ips...)
}

func grantCheckoutAccess(api sdkapi.IPluginApi, clnt sdkapi.IClientDevice) error {
    return api.Firewall().AllowClientToDstIpGroup(
        sdkapi.DstIpGroupClient{
            MacAddr:  clnt.MacAddr(),
            Ipv4Addr: clnt.Ipv4Addr(),
            Ipv6Addr: clnt.Ipv6Addr(),
        },
        paymentGatewayGroup,
        300, // revoke automatically after 5 minutes
    )
}
```

### Example: open a protocol/port for everyone pre-auth (service port)

```go
func createDnsServicePort(api sdkapi.IPluginApi) error {
    exists, err := api.Firewall().ServicePortExists("dns")
    if err != nil {
        return err
    }
    if exists {
        return nil
    }
    return api.Firewall().CreateServicePort("dns", []string{"udp", "tcp"}, 53)
}
```

`CreateServicePort` is safe to call from `OnReady` on every boot, even though the
in-memory `ServicePortExists` state resets on every process restart — it
transparently flushes any leftover kernel-side chain/set state from a prior
process run before rebuilding, so repeated boots never duplicate rules.
Checking `ServicePortExists` first is still recommended to avoid the
"already exists" error on a genuine second call *within the same process
run* (e.g. two plugins racing to create the same name).

### Example: allow or block a specific MAC

```go
if err := api.Firewall().AllowMAC(mac); err != nil {
    api.Logger().Error("firewall: allow mac failed: " + err.Error())
}

// ...later, an absolute deny regardless of session or whitelist state:
if err := api.Firewall().BlockMAC(mac); err != nil {
    api.Logger().Error("firewall: block mac failed: " + err.Error())
}
```

### Example: attaching a fully custom chain

Use this only when dst IP groups / service ports / MAC control genuinely don't cover the
case — e.g. your own DNAT target, or matching on something none of the built-in
primitives expose. Wire the jump once the shared table exists, then own every rule
inside your chain via your own `nft` calls:

```go
func Init(api sdkapi.IPluginApi) error {
    api.Network().OnReady(func() {
        if err := api.Firewall().AddForwardChainBeforeInternet("pppoe_forward"); err != nil {
            api.Logger().Error("firewall: wire forward chain: " + err.Error())
        }
        if err := api.Firewall().AddPreRoutingChainBeforeInternet("pppoe_prerouting"); err != nil {
            api.Logger().Error("firewall: wire prerouting chain: " + err.Error())
        }
        // From here, populate rules inside "pppoe_forward"/"pppoe_prerouting" directly
        // with your own nft calls — the SDK only created the chain and the jump.
    })
    return nil
}
```

"Before" placement is required here specifically because core's forward chain already
has a transparency-passthrough rule that accepts all traffic on unmanaged interfaces
(like `ppp*`) — a chain attached "after" would never see that traffic at all.

## Grants are ephemeral — persist and re-apply on boot

Every grant described above (`AllowMAC`, `BlockMAC`, dst IP groups, service ports, and
their client allow-lists) lives only in the running nftables table — it does **not**
survive a reboot or firewall reset. The pattern used across core plugins is:

1. Store the grant in your plugin's own DB table.
2. Re-create groups/ports and re-apply MAC grants from that table in `OnReady`.

```go
func Init(api sdkapi.IPluginApi) error {
    api.Network().OnReady(func() {
        if err := createDnsServicePort(api); err != nil {
            api.Logger().Error("firewall: reapply dns service port failed: " + err.Error())
        }
        for _, mac := range loadWhitelistedMacsFromDB(api) {
            if err := api.Firewall().AllowMAC(mac); err != nil {
                api.Logger().Error("firewall: reapply allow mac failed: " + err.Error())
            }
        }
    })
    return nil
}
```

`OnReady`, not `Init()`, is required here: the shared `inet internet` table (and, for
`AllowMAC`/`BlockMAC`, the `whitelist_macs`/`blocked_macs` sets) don't exist until
`nftables.Setup()` has run, which happens after every plugin's `Init()` during boot —
see the `OnReady` danger box at the top of [IFirewallAPI](../api/firewall-api.md).

No `plugin.json` field is required to use `api.Firewall()` — it's available to every
plugin through `sdkapi.IPluginApi` like any other core API.

## See also

- [IFirewallAPI](../api/firewall-api.md) — full method reference, parameters, and
  precedence rules for MAC access control.
- [IClientDevice](../api/client-device.md) — source of the `MacAddr`/`Ipv4Addr`/`Ipv6Addr`
  values used to build a `DstIpGroupClient`.
