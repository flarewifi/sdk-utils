// Package netmon provides the core's online monitor: a lightweight background
// service that polls for internet reachability and emits EventInternetUp /
// EventInternetDown on every transition. Other parts of the core (and plugins,
// via IEventsApi.OnInternetEvent) subscribe to drive network-dependent work —
// most importantly installing a plugin's system_packages and running its
// install scripts, which can only succeed once the device is actually online.
package netmon

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"core/internal/events"

	sdkapi "sdk/api"
)

// Default polling cadence and per-probe dial timeout. The probe is a cheap TCP
// connect to public DNS resolvers (port 53), so polling every few seconds is
// inexpensive and detects connectivity changes promptly.
const (
	defaultInterval = 10 * time.Second
	dialTimeout     = 3 * time.Second
)

// probeHosts are well-known, highly-available anycast resolvers reached by raw
// IP (no DNS lookup needed) on TCP/53. A successful connect to ANY of them means
// the device has working internet egress.
var probeHosts = []string{
	"1.1.1.1:53",
	"8.8.8.8:53",
	"9.9.9.9:53",
}

// active points at the running monitor, set when Start is called, so package-level
// callers (e.g. IMachineApi.IsOnline) can query connectivity without a handle on
// the Monitor. There is exactly one monitor per process.
var active atomic.Pointer[Monitor]

// IsOnline reports the last connectivity state observed by the running monitor.
// It returns false when the monitor has not started or has not completed its
// first probe yet — i.e. connectivity is treated as "not known to be up".
func IsOnline() bool {
	m := active.Load()
	return m != nil && m.IsUp()
}

// Monitor polls internet reachability and emits connectivity events.
type Monitor struct {
	events   *events.EventsManager
	logger   sdkapi.ILoggerApi
	interval time.Duration

	// up holds the last observed state. It starts as "unknown" (never probed) so
	// the very first probe always emits, giving subscribers an initial signal.
	up      atomic.Bool
	probed  atomic.Bool
	started atomic.Bool
}

// NewMonitor constructs an online monitor wired to the events manager and logger.
func NewMonitor(em *events.EventsManager, logger sdkapi.ILoggerApi) *Monitor {
	return &Monitor{
		events:   em,
		logger:   logger,
		interval: defaultInterval,
	}
}

// Start launches the polling loop in its own goroutine. It is safe to call once;
// subsequent calls are no-ops. The loop runs until ctx is cancelled.
func (m *Monitor) Start(ctx context.Context) {
	if !m.started.CompareAndSwap(false, true) {
		return
	}
	active.Store(m)

	go func() {
		// Probe immediately so a device that boots already-online provisions
		// without waiting a full interval, then settle into periodic polling.
		m.check(ctx)

		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.check(ctx)
			}
		}
	}()
}

// IsUp reports the last observed connectivity state. It returns false until the
// first probe has completed.
func (m *Monitor) IsUp() bool {
	return m.probed.Load() && m.up.Load()
}

// WaitOnline blocks until a connectivity probe succeeds or timeout elapses,
// returning true only if the machine became reachable. The first probe runs
// immediately (an already-online machine returns at once); thereafter it re-probes
// every interval until the deadline. ctx cancellation aborts the wait.
//
// It is used to gate online-only boot work (a plugin's system_packages/install
// scripts): the boot sequence waits a bounded time for internet so the booting
// page can show the install phase, then falls back to offline-first boot if the
// link never appears. It is independent of any running Monitor, so it works during
// boot before the monitor's polling loop has started.
func WaitOnline(ctx context.Context, timeout, interval time.Duration) bool {
	if probe() {
		return true
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-timer.C:
			return false
		case <-ticker.C:
			if probe() {
				return true
			}
		}
	}
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// check runs one probe and emits an event only when the state changes (or on the
// first probe). Emission is synchronous per the events contract; subscribers that
// do slow work spawn their own goroutine, so this does not stall the loop.
func (m *Monitor) check(ctx context.Context) {
	online := probe()

	prev := m.up.Load()
	first := !m.probed.Swap(true)
	m.up.Store(online)

	if !first && online == prev {
		return // no transition
	}

	event := sdkapi.EventInternetDown
	if online {
		event = sdkapi.EventInternetUp
	}

	if m.logger != nil {
		_ = m.logger.Info(fmt.Sprintf("online monitor: internet is %s", stateName(online)))
	}

	if err := m.events.EmitInternetEvent(ctx, event); err != nil && m.logger != nil {
		_ = m.logger.Error(fmt.Sprintf("online monitor: %s handler error: %v", event, err))
	}
}

// probe returns true if a TCP connection to any probe host succeeds within the
// dial timeout.
func probe() bool {
	for _, host := range probeHosts {
		conn, err := net.DialTimeout("tcp", host, dialTimeout)
		if err == nil {
			_ = conn.Close()
			return true
		}
	}
	return false
}

func stateName(up bool) string {
	if up {
		return "up"
	}
	return "down"
}
