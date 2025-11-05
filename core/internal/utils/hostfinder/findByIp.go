//go:build !dev

package hostfinder

import (
	"errors"
	"strings"

	"core/internal/utils/arp"
)

func FindByIp(ip string) (*HostData, error) {
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
