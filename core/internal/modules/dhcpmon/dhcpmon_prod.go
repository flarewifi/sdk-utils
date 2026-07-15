//go:build !dev

package dhcpmon

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"core/internal/modules/uci"
	cmd "core/utils/shell"
)

const (
	// dnsmasqSection is the anonymous UCI section holding dnsmasq's instance-wide
	// options in /etc/config/dhcp. It must be addressed by its unnamed selector
	// (@dnsmasq[0]); a lookup by the literal name "dnsmasq" never matches. Mirrors
	// core/internal/modules/captivedns's dnsmasqSection constant.
	dnsmasqSection = "@dnsmasq[0]"

	// scriptPath is where the dhcp-script is written. dnsmasq execs it on every
	// lease add/old/del; its only job is to forward the event into eventFifoPath.
	scriptPath = "/tmp/flare_dhcp_script.sh"

	// eventFifoPath is a named pipe the dhcp-script appends to and this package
	// tails. A FIFO (rather than a plain file) means events are consumed as they
	// arrive with no unbounded growth or rotation to manage — same approach as
	// core/internal/modules/ubus's hostapd_cli bridge.
	eventFifoPath = "/tmp/flare_dhcp_events"
)

// scriptContent forwards the dhcp-script's positional args and the DNSMASQ_*
// env vars this package needs into the FIFO, pipe-delimited so an empty hostname
// or tag list doesn't shift later fields (unlike whitespace-delimited output).
// DNSMASQ_TIME_REMAINING (seconds until expiry) is used instead of
// DNSMASQ_LEASE_EXPIRES because the latter is only set when dnsmasq was built
// against a working RTC.
const scriptContent = `#!/bin/sh
printf '%s|%s|%s|%s|%s|%s|%s\n' "$1" "$2" "$3" "$4" "${DNSMASQ_INTERFACE}" "${DNSMASQ_TIME_REMAINING}" "${DNSMASQ_TAGS}" >> ` + eventFifoPath + `
`

// Start writes the dhcp-script, points dnsmasq at it via UCI, and begins reading
// lease events from the FIFO. Safe to call once; subsequent calls are no-ops.
// ctx is not retained past setup — the read loop runs for the process lifetime,
// same as netmon's monitor and the ubus wifi-event bridge.
func (m *Monitor) Start(ctx context.Context) {
	if !m.started.CompareAndSwap(false, true) {
		return
	}

	if err := m.setup(); err != nil {
		m.logError(fmt.Sprintf("dhcpmon: setup failed: %v", err))
		return
	}

	go m.readEvents()
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// setup writes the dhcp-script, creates the FIFO, and points dnsmasq's
// dhcpscript option at the script, committing and restarting dnsmasq so the new
// option takes effect.
func (m *Monitor) setup() error {
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("write dhcp-script: %w", err)
	}

	os.Remove(eventFifoPath)
	if err := syscall.Mkfifo(eventFifoPath, 0666); err != nil {
		return fmt.Errorf("create dhcp event fifo: %w", err)
	}

	if ok := uci.UciTree.Set("dhcp", dnsmasqSection, "dhcpscript", scriptPath); !ok {
		return fmt.Errorf("set dhcp.%s.dhcpscript", dnsmasqSection)
	}
	if err := uci.UciTree.Commit(); err != nil {
		return fmt.Errorf("uci commit dhcp: %w", err)
	}

	// A restart (not reload/SIGHUP) is required: dhcp-script is only read by
	// dnsmasq at exec time, unlike address/dhcp_option which dnsmasq re-reads on
	// SIGHUP. Existing leases survive since they live in leasefile, not memory.
	if err := cmd.Exec("service dnsmasq restart", nil); err != nil {
		return fmt.Errorf("dnsmasq restart: %w", err)
	}

	return nil
}

// readEvents tails the FIFO for the process lifetime, parsing and emitting each
// line. Opening O_RDWR (rather than O_RDONLY) keeps a reader held open across
// writer churn, so the dhcp-script's blocking open-for-append never stalls
// waiting for a reader — same technique as the ubus hostapd_cli bridge.
func (m *Monitor) readEvents() {
	file, err := os.OpenFile(eventFifoPath, os.O_RDWR, 0666)
	if err != nil {
		m.logError(fmt.Sprintf("dhcpmon: open event fifo: %v", err))
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			m.logError(fmt.Sprintf("dhcpmon: read event fifo: %v", err))
			return
		}

		event, data, ok := parseLeaseLine(strings.TrimRight(line, "\n"), time.Now())
		if !ok {
			continue
		}

		if err := m.events.EmitDhcpEvent(context.Background(), event, data); err != nil {
			m.logError(fmt.Sprintf("dhcpmon: %s handler error: %v", event, err))
		}
	}
}

func (m *Monitor) logError(msg string) {
	if m.logger != nil {
		_ = m.logger.Error(msg)
	}
}
