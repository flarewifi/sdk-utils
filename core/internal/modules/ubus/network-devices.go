//go:build !dev

package ubus

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/goccy/go-json"

	cmd "core/utils/shell"
)

const sysNetPath = "/sys/class/net"

// GetNetworkDevices returns every device under /sys/class/net, enriched with
// ubus's per-device status where available. `ubus call network.device
// status` only reports devices netifd already manages (referenced by an
// interface, a bridge, or a `config device` UCI section) — a physical port
// nobody has wired into an interface yet (e.g. a spare LAN port) is invisible
// to it, so it's merged in here directly from sysfs instead of being silently
// dropped.
func GetNetworkDevices() ([]*NetworkDevice, error) {
	var out bytes.Buffer
	err := cmd.ExecOutput("ubus call network.device status", &out)
	if err != nil {
		return nil, err
	}

	var devStat map[string]*NetworkDevice
	err = json.Unmarshal(out.Bytes(), &devStat)
	if err != nil {
		return nil, err
	}
	for k := range devStat {
		devStat[k].Name = k
	}

	entries, err := os.ReadDir(sysNetPath)
	if err != nil {
		return nil, err
	}

	var devices []*NetworkDevice
	for _, entry := range entries {
		name := entry.Name()
		if dev, ok := devStat[name]; ok {
			devices = append(devices, dev)
			continue
		}
		devices = append(devices, sysfsNetworkDevice(name))
	}

	return devices, nil
}

func GetNetworkDevice(device string) (*NetworkDevice, error) {
	var out bytes.Buffer
	err := cmd.ExecOutput(fmt.Sprintf(`ubus call network.device status '{"name":"%s"}'`, device), &out)
	if err != nil {
		// Not managed by netifd (e.g. an unclaimed physical port) — ubus has
		// nothing on it, so fall back to reading it straight from sysfs.
		if _, statErr := os.Stat(filepath.Join(sysNetPath, device)); statErr != nil {
			return nil, err
		}
		return sysfsNetworkDevice(device), nil
	}

	var dev NetworkDevice
	if err := json.Unmarshal(out.Bytes(), &dev); err != nil {
		return nil, err
	}
	dev.Name = device
	return &dev, nil
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// sysfsNetworkDevice builds a NetworkDevice by reading /sys/class/net/<name>
// directly, for devices ubus doesn't manage (so it has no devtype/carrier/
// bridge-members data to report). Best-effort: unreadable attributes are left
// at their zero value rather than failing the whole listing.
func sysfsNetworkDevice(name string) *NetworkDevice {
	dev := &NetworkDevice{Name: name}
	base := filepath.Join(sysNetPath, name)

	if mac, err := os.ReadFile(filepath.Join(base, "address")); err == nil {
		dev.MacAddr = strings.TrimSpace(string(mac))
	}
	if state, err := os.ReadFile(filepath.Join(base, "operstate")); err == nil {
		dev.Up = strings.TrimSpace(string(state)) == "up"
	}
	if carrier, err := os.ReadFile(filepath.Join(base, "carrier")); err == nil {
		dev.Carrier = strings.TrimSpace(string(carrier)) == "1"
	}

	dev.Type = sysfsDeviceType(base)
	if dev.Type == "bridge" {
		if members, err := os.ReadDir(filepath.Join(base, "brif")); err == nil {
			for _, m := range members {
				dev.BridgeMembers = append(dev.BridgeMembers, m.Name())
			}
		}
	}

	if rx, err := os.ReadFile(filepath.Join(base, "statistics", "rx_bytes")); err == nil {
		if n, err := strconv.ParseUint(strings.TrimSpace(string(rx)), 10, 64); err == nil {
			dev.Stats.RxBytes = uint(n)
		}
	}
	if tx, err := os.ReadFile(filepath.Join(base, "statistics", "tx_bytes")); err == nil {
		if n, err := strconv.ParseUint(strings.TrimSpace(string(tx)), 10, 64); err == nil {
			dev.Stats.TxBytes = uint(n)
		}
	}

	return dev
}

// sysfsDeviceType best-effort classifies a device ubus has no devtype for,
// using the same "bridge"/"wlan"/"vlan"/"ethernet" values ubus's own devtype
// field uses (see sdkapi.NetDevType).
func sysfsDeviceType(base string) string {
	if _, err := os.Stat(filepath.Join(base, "bridge")); err == nil {
		return "bridge"
	}
	if _, err := os.Stat(filepath.Join(base, "wireless")); err == nil {
		return "wlan"
	}
	if uevent, err := os.ReadFile(filepath.Join(base, "uevent")); err == nil {
		if strings.Contains(string(uevent), "DEVTYPE=vlan") {
			return "vlan"
		}
	}
	return "ethernet"
}
