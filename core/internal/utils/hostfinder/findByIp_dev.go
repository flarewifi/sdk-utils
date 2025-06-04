//go:build dev

package hostfinder

func FindByIp(ip string) (*HostData, error) {
	return &HostData{
		IpAddr:   ip,
		MacAddr:  "00:00:00:00:00:00",
		Hostname: "localhost",
	}, nil
}
