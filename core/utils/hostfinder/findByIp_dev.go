//go:build dev

package hostfinder

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type HostInfo struct {
	MacAddr  string `json:"mac_addr"`
	Hostname string `json:"hostname"`
}

func GetHostFromRequest(r *http.Request) (*HostData, error) {
	// Get IP from ip_addr cookie in dev
	defaultIP := "10.0.0.10"

	var ipAddr string
	ipCookie, err := r.Cookie("ip_addr")

	if err != nil {
		ipAddr = defaultIP
	} else if ipCookie == nil || ipCookie.Value == "" {
		ipAddr = defaultIP
	} else {
		ipAddr = ipCookie.Value
	}

	// Read hosts.json to get mac_addr and hostname
	hostsData, err := os.ReadFile(filepath.Join(sdkutils.PathAppDir, "hosts.json"))
	if err != nil {
		// Fallback to defaults if hosts.json can't be read
		return &HostData{
			IpAddr:   ipAddr,
			MacAddr:  "00:00:00:00:00:00",
			Hostname: "localhost",
		}, nil
	}

	var hosts map[string]HostInfo
	if err := json.Unmarshal(hostsData, &hosts); err != nil {
		// Fallback to defaults if JSON parsing fails
		return &HostData{
			IpAddr:   ipAddr,
			MacAddr:  "00:00:00:00:00:00",
			Hostname: "localhost",
		}, nil
	}

	// Look up the IP in hosts.json
	if hostInfo, found := hosts[ipAddr]; found {
		return &HostData{
			IpAddr:   ipAddr,
			MacAddr:  hostInfo.MacAddr,
			Hostname: hostInfo.Hostname,
		}, nil
	}

	// Sentinel: "0.0.0.0" simulates a complete ARP failure (no MAC resolvable).
	// This allows UI testing of the fingerprint-hash fallback path in portal registration.
	if ipAddr == "0.0.0.0" {
		return &HostData{IpAddr: ipAddr, MacAddr: "", Hostname: ""}, nil
	}

	// Fallback to defaults if IP not found in hosts.json
	return &HostData{
		IpAddr:   ipAddr,
		MacAddr:  "00:00:00:00:00:00",
		Hostname: "localhost",
	}, nil
}
