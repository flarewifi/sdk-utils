//go:build !dev

package hostfinder

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"core/internal/utils/arp"
)

func GetHostFromRequest(r *http.Request) (*HostData, error) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, err
	}

	mac, ok := arp.Search(ip)
	if !ok {
		return nil, errors.New("cannot find host with IP: " + ip)
	}

	hostname, err := FindHostname(mac)
	if err != nil {
		return nil, err
	}

	return &HostData{
		MacAddr:  strings.ToUpper(mac),
		IpAddr:   ip,
		Hostname: hostname,
	}, nil
}
