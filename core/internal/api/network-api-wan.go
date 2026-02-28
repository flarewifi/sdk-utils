//go:build !dev

package api

import (
	"fmt"

	sdkapi "sdk/api"
)

// GetWanInterface returns the WAN network interface.
// In production (OpenWRT), it searches for standard WAN interface names.
func (self *NetworkApi) GetWanInterface() (sdkapi.INetworkInterface, error) {
	// Priority order for WAN interface names on OpenWRT
	wanNames := []string{"wan", "wan6", "wan0"}

	for _, name := range wanNames {
		iface, err := self.GetInterface(name)
		if err == nil {
			return iface, nil
		}
	}

	return nil, fmt.Errorf("no WAN interface found")
}
