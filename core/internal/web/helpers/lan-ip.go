package helpers

import (
	"core/internal/network"
	"net"
	"net/http"
)

func GetLanIP(r *http.Request) string {
	var lanIP string

	// Get client IP from request
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// Fallback: use request host
		lanIP = r.Host
	} else {
		// Find LAN interface by client IP
		lan, err := network.FindByIp(ip)
		if err != nil {
			// Fallback: use request host
			lanIP = r.Host
		} else {
			// Get LAN IP address
			lanIPAddr, err := lan.GetInterface().IpV4Addr()
			if err != nil {
				// Fallback: use request host
				lanIP = r.Host
			} else {
				lanIP = lanIPAddr.Addr
			}
		}
	}

	return lanIP
}
