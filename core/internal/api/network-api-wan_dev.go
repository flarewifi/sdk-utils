//go:build dev

package api

import (
	"fmt"
	"net"

	cnet "core/internal/network"
	sdkapi "sdk/api"
)

// GetWanInterface returns the WAN network interface.
// In dev mode, it returns the container's default network interface
// (the one with the default gateway route).
func (self *NetworkApi) GetWanInterface() (sdkapi.INetworkInterface, error) {
	// Get the default gateway interface by checking which interface
	// would be used to reach an external IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, fmt.Errorf("failed to determine default interface: %v", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	// Find the interface that has this IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %v", err)
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil && ip.Equal(localAddr.IP) {
				// Found the interface - return it wrapped as INetworkInterface
				return cnet.NewNetworkInterface(iface.Name), nil
			}
		}
	}

	return nil, fmt.Errorf("no WAN interface found")
}
