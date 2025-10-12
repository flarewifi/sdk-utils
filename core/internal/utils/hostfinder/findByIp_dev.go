//go:build dev

package hostfinder

import "fmt"

func FindByIp(ip string) (*HostData, error) {
	fmt.Println("[DEV] Finding host by IP:", ip)

	return &HostData{
		IpAddr:   ip,
		MacAddr:  "00:00:00:00:00:00",
		Hostname: "localhost",
	}, nil
}
