package jobs

import (
	"context"

	"core/internal/api"
	"core/internal/modules/dhcpmon"
)

// StartDhcpMonitor starts the DHCP lease-event listener, which hooks into
// dnsmasq's dhcp-script and forwards DHCPv4 lease add/old/del events to
// IEventsApi.OnDhcpEvent subscribers. This should be called once during
// application boot.
func StartDhcpMonitor(g *api.CoreGlobals) {
	monitor := dhcpmon.NewMonitor(g.EventsMgr, g.CoreAPI.Logger())
	monitor.Start(context.Background())
}
