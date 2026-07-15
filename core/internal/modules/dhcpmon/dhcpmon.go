// Package dhcpmon bridges dnsmasq's dhcp-script hook into the core's unified
// event system, so plugins can subscribe to DHCPv4 lease changes via
// IEventsApi.OnDhcpEvent. It points dnsmasq's dhcpscript UCI option at a small
// shell script that forwards every invocation into a FIFO, which this package
// reads and turns into EventDhcpLeaseAdd/Old/Del events — mirroring how
// core/internal/modules/ubus bridges hostapd_cli's action-script output.
//
// IPv6 leases on this machine are served by odhcpd, not dnsmasq (see
// data/openwrt-files/etc/config/dhcp's separate `config odhcpd` block and its own
// leasetrigger, already pointed at the stock /usr/sbin/odhcpd-update), so this
// package only ever emits DHCPv4 events.
package dhcpmon

import (
	"strings"
	"sync/atomic"
	"time"

	"core/internal/events"

	sdkapi "sdk/api"
)

// Monitor bridges dnsmasq's dhcp-script hook to DhcpEvent emissions.
type Monitor struct {
	events  *events.EventsManager
	logger  sdkapi.ILoggerApi
	started atomic.Bool
}

// NewMonitor constructs a DHCP lease-event listener wired to the events manager
// and logger. Call Start to begin listening.
func NewMonitor(em *events.EventsManager, logger sdkapi.ILoggerApi) *Monitor {
	return &Monitor{
		events: em,
		logger: logger,
	}
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// parseLeaseLine turns one pipe-delimited line written by the dhcp-script
// (action|mac|ip|hostname|interface|time_remaining|tags) into a DhcpEvent and its
// DhcpEventData. ok is false for a malformed line or an action outside
// add/old/del (dnsmasq only ever invokes our script with these three for DHCPv4
// lease changes; any other value is treated as unrecognized rather than guessed at).
func parseLeaseLine(line string, now time.Time) (event sdkapi.DhcpEvent, data sdkapi.DhcpEventData, ok bool) {
	fields := strings.Split(line, "|")
	if len(fields) != 7 {
		return "", sdkapi.DhcpEventData{}, false
	}

	switch fields[0] {
	case "add":
		event = sdkapi.EventDhcpLeaseAdd
	case "old":
		event = sdkapi.EventDhcpLeaseOld
	case "del":
		event = sdkapi.EventDhcpLeaseDel
	default:
		return "", sdkapi.DhcpEventData{}, false
	}

	data = sdkapi.DhcpEventData{
		Mac:       fields[1],
		Ip:        fields[2],
		Hostname:  fields[3],
		Interface: fields[4],
		Tags:      fields[6],
	}

	if remaining, err := time.ParseDuration(fields[5] + "s"); err == nil && remaining > 0 {
		data.LeaseExpires = now.Add(remaining)
	}

	return event, data, true
}
