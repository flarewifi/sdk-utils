package networkutil

import "net"

func IpInSubnet(ip string, networkIp string) (bool, error) {
	testIP := net.ParseIP(ip)
	_, subnet, err := net.ParseCIDR(networkIp)
	if err != nil {
		return false, err
	}
	return subnet.Contains(testIP), nil
}
