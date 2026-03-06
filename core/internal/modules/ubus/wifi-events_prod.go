//go:build !dev

package ubus

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	sdkapi "sdk/api"
)

const (
	// hostapdSocketDir is the directory where hostapd creates control sockets
	hostapdSocketDir = "/var/run/hostapd"

	// Temporary files for hostapd_cli action mode
	actionScriptPath = "/tmp/flare_hostapd_action.sh"
	eventFifoPath    = "/tmp/flare_hostapd_events"

	// reconnectDelay is the delay between reconnection attempts
	reconnectDelay = 5 * time.Second

	// scanInterval is the interval for scanning for new interfaces
	scanInterval = 30 * time.Second
)

// Action script content - hostapd_cli passes event data as arguments
// Format: $1=interface, $2=event_with_data
// We capture all arguments with $* to get the full event string
const actionScriptContent = `#!/bin/sh
echo "$*" >> /tmp/flare_hostapd_events
`

// Regex to parse event lines from FIFO:
// Format varies based on hostapd_cli mode:
// - Global mode: "global IFNAME=phy0-ap0 <3>AP-STA-CONNECTED 9e:e7:49:b4:0a:20 auth_alg=open"
// - Interface mode: "phy0-ap0 AP-STA-CONNECTED 9e:e7:49:b4:0a:20 auth_alg=open"
// We extract: interface name, event type, MAC address
// Note: Global mode has <3> prefix on event, interface mode does not
var eventRegex = regexp.MustCompile(`(?:global\s+)?(?:IFNAME=)?(\S+)\s+(?:<\d+>)?(AP-STA-(?:CONNECTED|DISCONNECTED))\s+([0-9a-fA-F:]{17})`)

// hostapdProcess tracks a running hostapd_cli process
type hostapdProcess struct {
	cmd           *exec.Cmd
	interfaceName string
}

// Start begins listening for WiFi client events from hostapd using hostapd_cli action mode.
// Also starts traffic-based fallback detection in parallel to catch missed events.
func (self *WifiMgr) Start() {
	log.Println("[WifiMgr] Starting WiFi event listener")

	// Always start the fallback detector in parallel
	// This ensures we catch disconnects even if hostapd_cli misses events
	self.startFallback()

	// Check if hostapd_cli is available
	if _, err := exec.LookPath("hostapd_cli"); err != nil {
		log.Printf("[WifiMgr] hostapd_cli not found, relying on fallback traffic-based detection only")
		return
	}

	// Check if hostapd sockets exist
	if _, err := os.Stat(hostapdSocketDir); os.IsNotExist(err) {
		log.Printf("[WifiMgr] hostapd socket directory not found (%s), relying on fallback traffic-based detection only", hostapdSocketDir)
		return
	}

	go self.run()
}

// startFallback starts the traffic-based fallback detector
func (self *WifiMgr) startFallback() {
	if self.trafficCh == nil {
		log.Println("[WifiMgr] WARNING: No traffic channel set, fallback detection disabled")
		return
	}

	detector := NewFallbackDetector(self, self.trafficCh)
	detector.Start()
}

// run is the main event loop
func (self *WifiMgr) run() {
	for {
		// Setup: create FIFO and action script
		if err := self.setup(); err != nil {
			log.Printf("[WifiMgr] Setup failed: %v, retrying in %v", err, reconnectDelay)
			time.Sleep(reconnectDelay)
			continue
		}

		// Start hostapd_cli process(es)
		processes, err := self.startHostapdCli()
		if err != nil {
			log.Printf("[WifiMgr] Failed to start hostapd_cli: %v, retrying in %v", err, reconnectDelay)
			self.cleanup()
			time.Sleep(reconnectDelay)
			continue
		}

		log.Printf("[WifiMgr] Listening for WiFi events (%d interface(s))", len(processes))

		// Read events from FIFO (blocks until FIFO is closed or error)
		self.readEvents()

		// Cleanup processes
		for _, p := range processes {
			self.stopProcess(p)
		}
		self.cleanup()

		log.Printf("[WifiMgr] Event reader exited, restarting in %v...", reconnectDelay)
		time.Sleep(reconnectDelay)
	}
}

// setup creates the FIFO and action script
func (self *WifiMgr) setup() error {
	// Remove old FIFO if exists
	os.Remove(eventFifoPath)

	// Create FIFO
	if err := syscall.Mkfifo(eventFifoPath, 0666); err != nil {
		return fmt.Errorf("create FIFO: %w", err)
	}

	// Write action script
	if err := os.WriteFile(actionScriptPath, []byte(actionScriptContent), 0755); err != nil {
		return fmt.Errorf("write action script: %w", err)
	}

	return nil
}

// cleanup removes temporary files
func (self *WifiMgr) cleanup() {
	os.Remove(eventFifoPath)
	os.Remove(actionScriptPath)
}

// startHostapdCli starts hostapd_cli process(es) in action mode
func (self *WifiMgr) startHostapdCli() ([]*hostapdProcess, error) {
	// Check if hostapd_cli is available
	if _, err := exec.LookPath("hostapd_cli"); err != nil {
		return nil, fmt.Errorf("hostapd_cli not found: %w", err)
	}

	var processes []*hostapdProcess

	// Try global interface first
	globalSocket := filepath.Join(hostapdSocketDir, "global")
	if _, err := os.Stat(globalSocket); err == nil {
		proc, err := self.startProcess("global")
		if err != nil {
			return nil, fmt.Errorf("start global: %w", err)
		}
		processes = append(processes, proc)
		return processes, nil
	}

	// Fallback: start process for each interface
	interfaces, err := self.scanHostapdInterfaces()
	if err != nil {
		return nil, fmt.Errorf("scan interfaces: %w", err)
	}

	if len(interfaces) == 0 {
		return nil, fmt.Errorf("no hostapd interfaces found")
	}

	for _, iface := range interfaces {
		proc, err := self.startProcess(iface)
		if err != nil {
			log.Printf("[WifiMgr] Failed to start process for %s: %v", iface, err)
			continue
		}
		processes = append(processes, proc)
	}

	if len(processes) == 0 {
		return nil, fmt.Errorf("failed to start any hostapd_cli processes")
	}

	return processes, nil
}

// startProcess starts a single hostapd_cli process for an interface
func (self *WifiMgr) startProcess(interfaceName string) (*hostapdProcess, error) {
	// Use -a for action script mode, -r for auto-reconnect
	cmd := exec.Command("hostapd_cli", "-i", interfaceName, "-a", actionScriptPath, "-r")

	// Redirect stderr to our stderr for debugging
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command: %w", err)
	}

	// Start goroutine to wait for process exit and log it
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("[WifiMgr] hostapd_cli for %s exited with error: %v", interfaceName, err)
		}
	}()

	return &hostapdProcess{
		cmd:           cmd,
		interfaceName: interfaceName,
	}, nil
}

// stopProcess stops a hostapd_cli process
func (self *WifiMgr) stopProcess(proc *hostapdProcess) {
	if proc.cmd != nil && proc.cmd.Process != nil {
		proc.cmd.Process.Kill()
	}
}

// readEvents reads events from the FIFO and emits them
func (self *WifiMgr) readEvents() {
	// Open FIFO for read-write to prevent EOF when writers close
	// Using O_RDWR keeps the FIFO open even when no writers are connected
	file, err := os.OpenFile(eventFifoPath, os.O_RDWR, 0666)
	if err != nil {
		log.Printf("[WifiMgr] Failed to open FIFO: %v", err)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("[WifiMgr] Read error: %v", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		self.parseAndEmitEvent(line)
	}
}

// scanHostapdInterfaces scans /var/run/hostapd/ for available control sockets
func (self *WifiMgr) scanHostapdInterfaces() ([]string, error) {
	entries, err := os.ReadDir(hostapdSocketDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var interfaces []string
	for _, entry := range entries {
		name := entry.Name()
		// Skip directories and the global socket (handled separately)
		if !entry.IsDir() && name != "global" {
			interfaces = append(interfaces, name)
		}
	}

	return interfaces, nil
}

// parseAndEmitEvent parses a line from the FIFO and emits WiFi events.
// Uses the shared state tracker to prevent duplicate events when both
// hostapd and fallback detection are active.
func (self *WifiMgr) parseAndEmitEvent(line string) {
	// Skip non-event lines
	if !strings.Contains(line, "AP-STA-") {
		return
	}

	matches := eventRegex.FindStringSubmatch(line)
	if matches == nil {
		log.Printf("[WifiMgr] Failed to parse event: %q", line)
		return
	}

	// Extract fields
	interfaceName := matches[1]
	eventType := matches[2]
	mac := strings.ToUpper(matches[3])

	var shouldEmit bool
	var event sdkapi.WifiClientEvent

	// Get state tracker (may be nil during early startup)
	stateTracker := self.stateTracker

	switch eventType {
	case "AP-STA-CONNECTED":
		if stateTracker != nil {
			shouldEmit = stateTracker.OnHostapdConnect(mac)
		} else {
			shouldEmit = true // No tracker yet, emit event
		}
		event = sdkapi.WifiEventClientConnected
		if shouldEmit {
			log.Printf("[WifiMgr-hostapd] Client CONNECTED on %s: %s", interfaceName, mac)
		} else {
			log.Printf("[WifiMgr-hostapd] Client CONNECTED on %s: %s (duplicate, suppressed)", interfaceName, mac)
		}

	case "AP-STA-DISCONNECTED":
		if stateTracker != nil {
			shouldEmit = stateTracker.OnHostapdDisconnect(mac)
		} else {
			shouldEmit = true // No tracker yet, emit event
		}
		event = sdkapi.WifiEventClientDisconnected
		if shouldEmit {
			log.Printf("[WifiMgr-hostapd] Client DISCONNECTED on %s: %s", interfaceName, mac)
		} else {
			log.Printf("[WifiMgr-hostapd] Client DISCONNECTED on %s: %s (already disconnected, suppressed)", interfaceName, mac)
		}

	default:
		return
	}

	// Only emit if state actually changed
	if shouldEmit {
		self.emit(WifiEvent{
			Interface: interfaceName,
			Mac:       mac,
			Event:     event,
		})
	}
}
