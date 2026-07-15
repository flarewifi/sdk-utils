//go:build !dev

package dhcpmon

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"core/internal/modules/uci"
	cmd "core/utils/shell"
)

const (
	// scriptPath is where the dhcp-script is written. dnsmasq execs it on every
	// lease add/old/del; its only job is to forward the event into eventFifoPath.
	scriptPath = "/tmp/flare_dhcp_script.sh"

	// eventFifoPath is a named pipe the dhcp-script appends to and this package
	// tails. A FIFO (rather than a plain file) means events are consumed as they
	// arrive with no unbounded growth or rotation to manage — same approach as
	// core/internal/modules/ubus's hostapd_cli bridge.
	eventFifoPath = "/tmp/flare_dhcp_events"

	// reconnectDelay is how long readEvents waits before reopening the FIFO after
	// a read error, so a transient hiccup doesn't spin-loop but also doesn't
	// permanently stop event delivery. Matches the ubus wifi-event bridge's
	// reconnectDelay.
	reconnectDelay = 5 * time.Second
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
// Setup (UCI writes + a blocking dnsmasq restart) runs in its own goroutine so a
// stalled service restart cannot stall the caller — jobs.Init() calls this
// synchronously during boot. ctx is not retained past setup — the read loop runs
// for the process lifetime, same as netmon's monitor and the ubus wifi-event
// bridge.
func (m *Monitor) Start(ctx context.Context) {
	if !m.started.CompareAndSwap(false, true) {
		return
	}

	go func() {
		if err := m.setup(); err != nil {
			m.logError(fmt.Sprintf("dhcpmon: setup failed: %v", err))
			return
		}
		m.run()
	}()
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// setup writes the dhcp-script, ensures the FIFO exists, and points dnsmasq's
// dhcpscript option at the script. The UCI write + dnsmasq restart are skipped
// entirely when dhcpscript is already set correctly, so re-running setup on every
// process restart (e.g. a dev reflex rebuild, or a crash-loop) doesn't force a
// disruptive dnsmasq restart when nothing actually changed.
func (m *Monitor) setup() error {
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("write dhcp-script: %w", err)
	}

	if err := syscall.Mkfifo(eventFifoPath, 0666); err != nil && !errors.Is(err, syscall.EEXIST) {
		return fmt.Errorf("create dhcp event fifo: %w", err)
	}

	current, _ := uci.UciTree.Get("dhcp", uci.DnsmasqSection, "dhcpscript")
	if len(current) == 1 && current[0] == scriptPath {
		return nil
	}

	if ok := uci.UciTree.Set("dhcp", uci.DnsmasqSection, "dhcpscript", scriptPath); !ok {
		return fmt.Errorf("set dhcp.%s.dhcpscript", uci.DnsmasqSection)
	}
	if err := uci.UciTree.Commit(); err != nil {
		// Set() already staged the change on the shared, process-lifetime
		// UciTree singleton; revert it so an unrelated later Commit() elsewhere
		// in the app can't silently flush this half-applied change to disk.
		uci.UciTree.Revert("dhcp")
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

// run tails the FIFO for the process lifetime. If readEvents returns (the FIFO
// read failed), it waits reconnectDelay and reopens it, so a transient I/O error
// pauses event delivery instead of permanently stopping it.
func (m *Monitor) run() {
	for {
		if err := m.readEvents(); err != nil {
			m.logError(fmt.Sprintf("dhcpmon: %v, reconnecting in %s", err, reconnectDelay))
		}
		time.Sleep(reconnectDelay)
	}
}

// readEvents opens the FIFO and parses/emits each line until a read error
// occurs, which it returns to the caller. Opening O_RDWR (rather than O_RDONLY)
// keeps a reader held open across writer churn, so the dhcp-script's blocking
// open-for-append never stalls waiting for a reader — same technique as the ubus
// hostapd_cli bridge.
func (m *Monitor) readEvents() error {
	file, err := os.OpenFile(eventFifoPath, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("open dhcp event fifo: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read dhcp event fifo: %w", err)
		}

		event, data, ok := parseLeaseLine(strings.TrimRight(line, "\n"), time.Now().UTC())
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
