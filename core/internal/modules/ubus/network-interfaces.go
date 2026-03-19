package ubus

import (
	"bufio"
	"bytes"
	"errors"
	"log"
	"net"
	"strings"

	"github.com/goccy/go-json"

	cmd "core/utils/shell"
)

type IpV4Addr struct {
	Addr    string `json:"address"`
	Netmask int    `json:"mask"`
}

// IpV6Addr holds a single IPv6 address and its prefix length as reported by UBUS.
type IpV6Addr struct {
	Addr      string `json:"address"`
	PrefixLen int    `json:"mask"`
}

type NetworkInterface struct {
	Device        string     `json:"device"`
	Up            bool       `json:"up"`
	IpV4Addresses []IpV4Addr `json:"ipv4-address"`
	IpV6Addresses []IpV6Addr `json:"ipv6-address"`
	DnsServers    []string   `json:"dns-server"`
}

func (iface *NetworkInterface) GetDevice() (*NetworkDevice, error) {
	return GetNetworkDevice(iface.Device)
}

func (iface *NetworkInterface) IpV4Addr() (ip IpV4Addr, err error) {
	if len(iface.IpV4Addresses) > 0 {
		return iface.IpV4Addresses[0], nil
	}
	return ip, errors.New("no IPv4 addresses found")
}

// IpV6Addr returns the preferred global-scope IPv6 address on the interface.
// Link-local addresses (fe80::/10) are skipped in favour of routable ones.
// Returns an error if no addresses are available or if only link-local
// addresses are present (link-locals are not usable for DNAT or routing).
func (iface *NetworkInterface) IpV6Addr() (ip IpV6Addr, err error) {
	if len(iface.IpV6Addresses) == 0 {
		return ip, errors.New("no IPv6 addresses found")
	}

	// Prefer global unicast addresses over link-local / other scopes.
	// Use net.ParseIP for correct, case-insensitive classification.
	for _, addr := range iface.IpV6Addresses {
		parsed := net.ParseIP(addr.Addr)
		if parsed != nil && !parsed.IsLinkLocalUnicast() {
			return addr, nil
		}
	}

	return ip, errors.New("only link-local IPv6 addresses found on interface")
}

func GetInterfaceNames() (names []string, err error) {
	var out strings.Builder
	err = cmd.ExecOutput("ubus list network.interface.*", &out)
	if err != nil {
		return names, err
	}

	outstr := strings.TrimSpace(out.String())
	scanner := bufio.NewScanner(strings.NewReader(outstr))

	list := []string{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			iface := strings.Replace(line, "network.interface.", "", 1)
			list = append(list, iface)
		}
	}

	return list, nil
}

func GetNetworkInterfaces() (map[string]*NetworkInterface, error) {
	list, err := GetInterfaceNames()
	if err != nil {
		return nil, err
	}

	ifMap := map[string]*NetworkInterface{}

	for _, iface := range list {
		log.Println("Get iface:", iface)
		ifaceData, err := GetNetworkInterface(iface)
		if err != nil {
			return nil, err
		}
		ifMap[iface] = ifaceData
	}

	return ifMap, nil
}

func GetNetworkInterface(iface string) (*NetworkInterface, error) {
	var out bytes.Buffer
	err := cmd.ExecOutput("ubus call network.interface."+iface+" status", &out)
	if err != nil {
		return nil, err
	}
	var ifaceData NetworkInterface
	if err := json.Unmarshal(out.Bytes(), &ifaceData); err != nil {
		return nil, err
	}
	return &ifaceData, nil
}
