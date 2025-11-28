//go:build dev

package hostfinder

import (
	"net/http"
)

func GetHostFromRequest(r *http.Request) (*HostData, error) {
	// Get IP from ip_addr cookie in dev
	defaultIP := "10.0.0.10"

	var ipAddr string
	ipCookie, err := r.Cookie("ip_addr")
	if err != nil {
		ipAddr = defaultIP
	}

	if ipCookie == nil {
		ipAddr = defaultIP
	} else {
		if ipCookie.Value == "" {
			ipAddr = defaultIP
		}
	}

	return &HostData{
		IpAddr:   ipAddr,
		MacAddr:  "00:00:00:00:00:00",
		Hostname: "localhost",
	}, nil
}
