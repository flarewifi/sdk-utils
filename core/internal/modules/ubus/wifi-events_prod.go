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
func (self *WifiMgr) Start() {
	log.Println("[WifiMgr] Starting WiFi event listener (hostapd_cli action mode)")
	log.Printf("[WifiMgr] Hostapd socket directory: %s", hostapdSocketDir)
	log.Printf("[WifiMgr] Action script: %s", actionScriptPath)
	log.Printf("[WifiMgr] Event FIFO: %s", eventFifoPath)

	go self.run()
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

		log.Printf("[WifiMgr] Started %d hostapd_cli process(es)", len(processes))

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
	log.Printf("[WifiMgr] Creating FIFO at %s", eventFifoPath)
	if err := syscall.Mkfifo(eventFifoPath, 0666); err != nil {
		return fmt.Errorf("create FIFO: %w", err)
	}

	// Write action script
	log.Printf("[WifiMgr] Writing action script to %s", actionScriptPath)
	if err := os.WriteFile(actionScriptPath, []byte(actionScriptContent), 0755); err != nil {
		return fmt.Errorf("write action script: %w", err)
	}

	return nil
}

// cleanup removes temporary files
func (self *WifiMgr) cleanup() {
	os.Remove(eventFifoPath)
	os.Remove(actionScriptPath)
	log.Println("[WifiMgr] Cleaned up temporary files")
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
		log.Println("[WifiMgr] Global socket found, using global interface")
		proc, err := self.startProcess("global")
		if err != nil {
			return nil, fmt.Errorf("start global: %w", err)
		}
		processes = append(processes, proc)
		return processes, nil
	}

	log.Println("[WifiMgr] Global socket not found, scanning for individual interfaces")

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
	log.Printf("[WifiMgr] Starting hostapd_cli -i %s -a %s", interfaceName, actionScriptPath)

	// Use -a for action script mode, -r for auto-reconnect
	cmd := exec.Command("hostapd_cli", "-i", interfaceName, "-a", actionScriptPath, "-r")

	// Redirect stderr to our stderr for debugging
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command: %w", err)
	}

	log.Printf("[WifiMgr] hostapd_cli for %s started (PID: %d)", interfaceName, cmd.Process.Pid)

	// Start goroutine to wait for process exit and log it
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("[WifiMgr] hostapd_cli for %s exited with error: %v", interfaceName, err)
		} else {
			log.Printf("[WifiMgr] hostapd_cli for %s exited normally", interfaceName)
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
		log.Printf("[WifiMgr] Stopping hostapd_cli for %s (PID: %d)", proc.interfaceName, proc.cmd.Process.Pid)
		proc.cmd.Process.Kill()
	}
}

// readEvents reads events from the FIFO and emits them
func (self *WifiMgr) readEvents() {
	log.Printf("[WifiMgr] Opening FIFO for reading: %s", eventFifoPath)

	// Open FIFO for read-write to prevent EOF when writers close
	// Using O_RDWR keeps the FIFO open even when no writers are connected
	file, err := os.OpenFile(eventFifoPath, os.O_RDWR, 0666)
	if err != nil {
		log.Printf("[WifiMgr] Failed to open FIFO: %v", err)
		return
	}
	defer file.Close()

	log.Println("[WifiMgr] FIFO opened, reading events...")

	reader := bufio.NewReader(file)
	eventCount := 0

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

		eventCount++
		log.Printf("[WifiMgr] Event #%d from FIFO: %q", eventCount, line)

		self.parseAndEmitEvent(line)
	}
}

// scanHostapdInterfaces scans /var/run/hostapd/ for available control sockets
func (self *WifiMgr) scanHostapdInterfaces() ([]string, error) {
	entries, err := os.ReadDir(hostapdSocketDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[WifiMgr] Hostapd socket directory %s does not exist yet", hostapdSocketDir)
			return nil, nil
		}
		return nil, err
	}

	log.Printf("[WifiMgr] Found %d entries in %s", len(entries), hostapdSocketDir)

	var interfaces []string
	for _, entry := range entries {
		name := entry.Name()
		log.Printf("[WifiMgr] Entry: %s (isDir: %v)", name, entry.IsDir())

		// Skip directories and the global socket (handled separately)
		if !entry.IsDir() && name != "global" {
			interfaces = append(interfaces, name)
		}
	}

	if len(interfaces) == 0 {
		log.Printf("[WifiMgr] No hostapd interface sockets found in %s", hostapdSocketDir)
	} else {
		log.Printf("[WifiMgr] Found %d hostapd interfaces: %v", len(interfaces), interfaces)
	}

	return interfaces, nil
}

// parseAndEmitEvent parses a line from the FIFO and emits WiFi events
func (self *WifiMgr) parseAndEmitEvent(line string) {
	// Skip non-event lines
	if !strings.Contains(line, "AP-STA-") {
		log.Printf("[WifiMgr] Skipping non-event line: %q", line)
		return
	}

	log.Printf("[WifiMgr] Parsing event: %q", line)

	matches := eventRegex.FindStringSubmatch(line)
	if matches == nil {
		log.Printf("[WifiMgr] Failed to parse event: %q", line)
		return
	}

	// Extract fields
	interfaceName := matches[1]
	eventType := matches[2]
	mac := strings.ToUpper(matches[3])

	log.Printf("[WifiMgr] Parsed - Interface: %s, Event: %s, MAC: %s", interfaceName, eventType, mac)

	var event sdkapi.WifiClientEvent
	switch eventType {
	case "AP-STA-CONNECTED":
		event = sdkapi.WifiEventClientConnected
		log.Printf("[WifiMgr] Client CONNECTED on %s: %s", interfaceName, mac)
	case "AP-STA-DISCONNECTED":
		event = sdkapi.WifiEventClientDisconnected
		log.Printf("[WifiMgr] Client DISCONNECTED on %s: %s", interfaceName, mac)
	default:
		log.Printf("[WifiMgr] Unknown event type: %s", eventType)
		return
	}

	self.emit(WifiEvent{
		Interface: interfaceName,
		Mac:       mac,
		Event:     event,
	})
}
