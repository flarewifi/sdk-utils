# IUciApi

The `IUciApi` provides access to the OpenWrt **Unified Configuration Interface (UCI)** — the system used to read and write OpenWrt network, DHCP, and wireless configuration. It exposes sub-APIs for network interfaces, bridge VLANs, DHCP, and wireless devices.

To get an instance of `IUciApi`:

```go
uciAPI := api.Uci()
```

---

## IUciApi Methods

### Uci

Returns the underlying raw UCI tree (`uci.Tree`) from the `github.com/digineo/go-uci` library. Use this for low-level UCI reads/writes not covered by the sub-APIs.

```go
tree := api.Uci().Uci()
// e.g. tree.Get("network", "lan", "ipaddr")
```

### Network

Returns `IUciNetworkApi` for reading and writing network device, bridge, bridge-VLAN, and interface configuration.

```go
netApi := api.Uci().Network()
```

### Dhcp

Returns `IDhcpApi` for reading and writing DHCP server configuration.

```go
dhcpApi := api.Uci().Dhcp()
```

### Wireless

Returns `IWirelessApi` for reading and writing wireless device and interface configuration.

```go
wirelessApi := api.Uci().Wireless()
```

---

## IUciNetworkApi Methods

Access via `api.Uci().Network()`.

### Device Methods

#### GetDevice

Returns the device name associated with a UCI network section.

```go
dev, err := api.Uci().Network().GetDevice("lan")
if err != nil {
    // handle error
}
fmt.Println("Device:", dev) // e.g. "br-lan"
```

#### GetDeviceSec

Returns the UCI section name for a given device name.

```go
section, err := api.Uci().Network().GetDeviceSec("br-lan")
if err != nil {
    // handle error
}
fmt.Println("Section:", section)
```

#### GetDeviceType

Returns the type of a network device section (e.g. `"bridge"`).

```go
devType, err := api.Uci().Network().GetDeviceType("lan")
if err != nil {
    // handle error
}
fmt.Println("Type:", devType) // e.g. "bridge"
```

### Bridge Methods

#### GetBridgeSecs

Returns a list of all UCI sections that represent bridge devices.

```go
sections := api.Uci().Network().GetBridgeSecs()
for _, sec := range sections {
    fmt.Println("Bridge section:", sec)
}
```

#### GetBridgeVlanFilter

Reports whether VLAN filtering is enabled on a bridge section.

```go
enabled := api.Uci().Network().GetBridgeVlanFilter("lan")
fmt.Println("VLAN filtering:", enabled)
```

#### GetBridgePorts

Returns the list of port names attached to a bridge.

```go
ports, err := api.Uci().Network().GetBridgePorts("lan")
if err != nil {
    // handle error
}
fmt.Println("Ports:", ports) // e.g. ["eth0", "eth1"]
```

#### SetBridgePorts

Sets the list of ports for a bridge section.

```go
err := api.Uci().Network().SetBridgePorts("lan", []string{"eth0", "eth1"})
if err != nil {
    // handle error
}
```

#### SetBridgeVlanFilter

Enables or disables VLAN filtering on a bridge section.

```go
err := api.Uci().Network().SetBridgeVlanFilter("lan", true)
if err != nil {
    // handle error
}
```

### Bridge-VLAN Methods

#### CreateBrVlan

Creates a new bridge-VLAN entry with the given ports.

```go
brvlan := &sdkapi.BrVlan{Device: "br-lan", VlanID: 10}
ports := []*sdkapi.BrVlanPort{
    {Device: "eth0", Tagged: true},
    {Device: "eth1", Untagged: true, Primary: true},
}
err := api.Uci().Network().CreateBrVlan(brvlan, ports)
if err != nil {
    // handle error
}
```

#### GetBrVlanSecs

Returns all UCI section names for bridge-VLAN entries.

```go
sections := api.Uci().Network().GetBrVlanSecs()
```

#### GetBrVlanSec

Returns the UCI section name for a specific bridge-VLAN, if it exists.

```go
brvlan := &sdkapi.BrVlan{Device: "br-lan", VlanID: 10}
section, ok := api.Uci().Network().GetBrVlanSec(brvlan)
if ok {
    fmt.Println("Section:", section)
}
```

#### GetBrVlanID

Returns the VLAN ID for a bridge-VLAN.

```go
brvlan := &sdkapi.BrVlan{Device: "br-lan", VlanID: 10}
vlanid, ok := api.Uci().Network().GetBrVlanID(brvlan)
if ok {
    fmt.Println("VLAN ID:", vlanid)
}
```

#### GetBrVlanPorts

Returns the list of ports configured on a bridge-VLAN.

```go
brvlan := &sdkapi.BrVlan{Device: "br-lan", VlanID: 10}
ports, err := api.Uci().Network().GetBrVlanPorts(brvlan)
if err != nil {
    // handle error
}
for _, p := range ports {
    fmt.Println("Port:", p.Device, "Tagged:", p.Tagged)
}
```

#### SetBrVlanPorts

Updates the port list for a bridge-VLAN.

```go
brvlan := &sdkapi.BrVlan{Device: "br-lan", VlanID: 10}
ports := []*sdkapi.BrVlanPort{
    {Device: "eth0", Tagged: true},
}
err := api.Uci().Network().SetBrVlanPorts(brvlan, ports)
```

#### DeleteBrVlan

Removes a bridge-VLAN UCI entry.

```go
brvlan := &sdkapi.BrVlan{Device: "br-lan", VlanID: 10}
api.Uci().Network().DeleteBrVlan(brvlan)
```

### Interface Methods

#### GetInterface

Returns the `INetIface` configuration for a UCI section name.

```go
iface, err := api.Uci().Network().GetInterface("lan")
if err != nil {
    // handle error
}
fmt.Printf("Device: %s, IP: %s\n", iface.Device, iface.IpAddr)
```

#### GetInterfaceSecs

Returns all UCI network interface section names.

```go
sections := api.Uci().Network().GetInterfaceSecs()
for _, sec := range sections {
    fmt.Println("Interface section:", sec)
}
```

#### GetInterfaces

Returns all network interfaces as `[]*INetIface`.

```go
ifaces, err := api.Uci().Network().GetInterfaces()
if err != nil {
    // handle error
}
for _, iface := range ifaces {
    fmt.Printf("Section: %s, Proto: %s, IP: %s\n", iface.Section, iface.Proto, iface.IpAddr)
}
```

#### SetInterface

Writes a network interface configuration to a UCI section.

```go
cfg := &sdkapi.INetIface{
    Device:  "eth0",
    Proto:   "static",
    IpAddr:  "192.168.2.1",
    Netmask: "255.255.255.0",
}
err := api.Uci().Network().SetInterface("lan2", cfg)
if err != nil {
    // handle error
}
```

---

## IDhcpApi Methods

Access via `api.Uci().Dhcp()`.

### GetSection

Returns the UCI section name for the DHCP server configured on a given interface name.

```go
section, ok := api.Uci().Dhcp().GetSection("br-lan")
if ok {
    fmt.Println("DHCP section:", section)
}
```

### GetConfig

Returns the DHCP configuration for a UCI section.

```go
section, ok := api.Uci().Dhcp().GetSection("br-lan")
if ok {
    cfg, ok := api.Uci().Dhcp().GetConfig(section)
    if ok {
        fmt.Printf("DHCP start: %s, limit: %d, lease: %dh\n", cfg.StartIp, cfg.Limit, cfg.LeaseHour)
    }
}
```

### SetConfig

Writes a new DHCP configuration for a given interface.

```go
cfg := &sdkapi.DhcpCfg{
    Ifname:    "br-lan",
    StartIp:   "192.168.1.100",
    Limit:     150,
    LeaseHour: 12,
}
err := api.Uci().Dhcp().SetConfig("br-lan", cfg)
if err != nil {
    // handle error
}
```

### GetDnsmasqLeasesFiles

Returns the list of dnsmasq lease file paths configured on the system.

```go
files, err := api.Uci().Dhcp().GetDnsmasqLeasesFiles()
if err != nil {
    // handle error
}
for _, f := range files {
    fmt.Println("Lease file:", f)
}
```

---

## IWirelessApi Methods

Access via `api.Uci().Wireless()`.

### GetDeviceSecs

Returns all UCI section names for wireless radio devices.

```go
sections := api.Uci().Wireless().GetDeviceSecs()
```

### GetIfaceSecs

Returns all UCI section names for wireless interfaces (VAPs).

```go
sections := api.Uci().Wireless().GetIfaceSecs()
```

### GetDevice

Returns the `WifiDev` configuration for a UCI section.

```go
dev, err := api.Uci().Wireless().GetDevice("radio0")
if err != nil {
    // handle error
}
fmt.Printf("Band: %s, Channel: %d\n", dev.Band, dev.Channel)
```

### GetIface

Returns the `WifiIface` configuration for a UCI section.

```go
iface, err := api.Uci().Wireless().GetIface("default_radio0")
if err != nil {
    // handle error
}
fmt.Printf("SSID: %s, Encryption: %s\n", iface.Ssid, iface.Encryption)
```

### GetDevices

Returns all wireless radio devices.

```go
devs := api.Uci().Wireless().GetDevices()
for _, d := range devs {
    fmt.Printf("Radio %s: band=%s channel=%d\n", d.Section, d.Band, d.Channel)
}
```

### GetIfaces

Returns all wireless interfaces (VAPs).

```go
ifaces := api.Uci().Wireless().GetIfaces()
for _, iface := range ifaces {
    fmt.Printf("VAP %s: SSID=%s\n", iface.Section, iface.Ssid)
}
```

### SetDevice

Writes a wireless radio device configuration.

```go
cfg := &sdkapi.WifiDev{
    Section: "radio0",
    Type:    "mac80211",
    Channel: 6,
    Band:    "2g",
    Htmode:  "HT20",
}
err := api.Uci().Wireless().SetDevice("radio0", cfg)
```

### SetIface

Writes a wireless interface (VAP) configuration.

```go
cfg := &sdkapi.WifiIface{
    Section:    "default_radio0",
    Device:     "radio0",
    Network:    "lan",
    Mode:       "ap",
    Ssid:       "MyHotspot",
    Encryption: "psk2",
    Key:        "mysecretkey",
}
err := api.Uci().Wireless().SetIface("default_radio0", cfg)
```

---

## Types

### INetIface

Represents a UCI network interface (section in `/etc/config/network`).

| Field | Type | Description |
|-------|------|-------------|
| `Section` | `string` | UCI section name (e.g. `"lan"`) |
| `Device` | `string` | Underlying network device (e.g. `"br-lan"`) |
| `Proto` | `string` | Protocol: `"static"`, `"dhcp"`, `"pppoe"`, etc. |
| `IpAddr` | `string` | IPv4 address |
| `Netmask` | `string` | Subnet mask |
| `Gateway` | `string` | Default gateway |

### IDevice

Interface representing a network device.

| Method | Return Type | Description |
|--------|-------------|-------------|
| `Name()` | `string` | Device name (e.g. `"br-lan"`) |
| `Type()` | `string` | Device type (e.g. `"bridge"`) |
| `BrPorts()` | `[]string` | Bridge port names (empty for non-bridge) |

### BrVlan

Identifies a bridge VLAN by device and VLAN ID. `String()` returns `"device.vlanid"` (e.g. `"br-lan.10"`).

```go
type BrVlan struct {
    Device string
    VlanID int
}
```

### BrVlanPort

Represents a port in a bridge-VLAN entry. `String()` formats the port as `"device:u*"` (untagged+primary), `"device:u"` (untagged), or `"device:t"` (tagged).

```go
type BrVlanPort struct {
    Device   string
    Tagged   bool
    Untagged bool
    Primary  bool
}
```

### DhcpCfg

DHCP server configuration for a network interface.

```go
type DhcpCfg struct {
    Section   string // UCI section name
    Ifname    string // Interface name (e.g. "br-lan")
    StartIp   string // First IP to lease (e.g. "192.168.1.100")
    Limit     uint   // Maximum number of leases
    LeaseHour uint   // Lease duration in hours
}
```

### WifiDev

Wireless radio device configuration.

```go
type WifiDev struct {
    Section  string // UCI section name (e.g. "radio0")
    Type     string // Driver type (e.g. "mac80211")
    Path     string // Hardware path
    Channel  uint   // Wireless channel number
    Band     string // Band: "2g", "5g", or "6g"
    Htmode   string // HT mode (e.g. "HT20", "VHT80")
    Disabled bool   // Whether the radio is disabled
}
```

### WifiIface

Wireless interface (VAP) configuration.

```go
type WifiIface struct {
    Section    string // UCI section name (e.g. "default_radio0")
    Device     string // Parent radio section (e.g. "radio0")
    Network    string // Network section to attach to (e.g. "lan")
    Mode       string // Operating mode: "ap", "sta", "monitor"
    Ssid       string // SSID (network name)
    Encryption string // Encryption type (see WifiEncryptions)
    Key        string // Passphrase or key
}
```

### Constants

#### WifiBands

Valid wireless band strings for `WifiDev.Band`:

```go
var WifiBands = []string{"2g", "5g", "6g"}
```

#### WifiEncryptions

Valid encryption strings for `WifiIface.Encryption` (source: [OpenWrt docs](https://openwrt.org/docs/guide-user/network/wifi/basic)):

```go
var WifiEncryptions = []string{
    "none", "sae", "sae-mixed", "psk2", "psk", "psk-mixed",
    "wep", "wep+open", "wep+shared",
    "wpa3", "wpa3-mixed", "wpa2", "wpa",
    // ... and many combination variants
}
```

---

## Error Types

These sentinel errors are returned by UCI network methods:

| Variable | Error Message | Meaning |
|----------|---------------|---------|
| `ErrNoDhcp` | `"DHCP server is not running on network interface."` | No DHCP section found for the interface |
| `ErrNotBridge` | `"Not a bridge device"` | The device section is not a bridge |
| `ErrNoVlanID` | `"No VLAN ID"` | No VLAN ID found in the bridge-VLAN section |
| `ErrNotBrVlan` | `"Not a bridge-vlan device"` | Device is not a bridge-VLAN type |
| `ErrEmptyDev(ifname)` | `"Can't get device for interface <ifname>"` | Interface has no device configured |

```go
devices, err := api.Uci().Network().GetBridgePorts("wan")
if err == sdkapi.ErrNotBridge {
    fmt.Println("WAN is not a bridge")
}
```

---

## Usage Example

### Reconfigure LAN IP and DHCP

```go
func reconfigureLan(api sdkapi.IPluginApi) error {
    netApi := api.Uci().Network()
    dhcpApi := api.Uci().Dhcp()

    // Update LAN interface IP
    err := netApi.SetInterface("lan", &sdkapi.INetIface{
        Device:  "br-lan",
        Proto:   "static",
        IpAddr:  "10.0.0.1",
        Netmask: "255.255.255.0",
    })
    if err != nil {
        return err
    }

    // Update DHCP range
    err = dhcpApi.SetConfig("br-lan", &sdkapi.DhcpCfg{
        Ifname:    "br-lan",
        StartIp:   "10.0.0.100",
        Limit:     100,
        LeaseHour: 24,
    })
    return err
}
```

### Change WiFi SSID

```go
func changeSSID(api sdkapi.IPluginApi, newSSID string) error {
    wirelessApi := api.Uci().Wireless()

    iface, err := wirelessApi.GetIface("default_radio0")
    if err != nil {
        return err
    }

    iface.Ssid = newSSID
    return wirelessApi.SetIface("default_radio0", iface)
}
```

---

## Related

- [IWifiApi](./wifi-api.md) - High-level WiFi hotspot management (portal SSID, captive portal)
- [INetworkApi](./network-api.md) - Runtime network device and interface information
- [IFirewallAPI](./firewall-api.md) - Firewall rule management
