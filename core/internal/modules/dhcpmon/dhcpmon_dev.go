//go:build dev

package dhcpmon

import "context"

// Start is a no-op in dev mode: dnsmasq doesn't run in the dev container, so
// there is no dhcp-script hook to wire up.
func (m *Monitor) Start(ctx context.Context) {
}
