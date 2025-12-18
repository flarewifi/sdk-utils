package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"core/tools/shell"
	sdkapi "sdk/api"
)

func NewFirewallApi(api *PluginApi) {
	firewallApi := &FirewallApi{
		activeTimers: make(map[string]*time.Timer),
		timersMutex:  &sync.Mutex{},
	}
	api.FirewallAPI = firewallApi
}

type FirewallApi struct {
	activeTimers map[string]*time.Timer // Track active removal timers by "destIP:mac" key
	timersMutex  *sync.Mutex            // Protect concurrent access to activeTimers
}

// NFTables JSON structure definitions
type nftablesOutput struct {
	Nftables []json.RawMessage `json:"nftables"`
}

type nftRule struct {
	Rule struct {
		Family string            `json:"family"`
		Table  string            `json:"table"`
		Chain  string            `json:"chain"`
		Handle int               `json:"handle"`
		Expr   []json.RawMessage `json:"expr"`
	} `json:"rule"`
}

type nftMatch struct {
	Match struct {
		Op    string          `json:"op"`
		Left  json.RawMessage `json:"left"`
		Right interface{}     `json:"right"`
	} `json:"match"`
}

type nftPayload struct {
	Payload struct {
		Protocol string `json:"protocol"`
		Field    string `json:"field"`
	} `json:"payload"`
}

// ResolveHostnameToIps is implemented in firewall-api-resolve.go and firewall-api-resolve_dev.go
// with different behavior for dev and production builds

// OpenIpForClientDevice opens firewall access for a specific client device to a destination IP
func (self *FirewallApi) OpenIpForClientDevice(params sdkapi.OpenIpForClientDeviceParams) error {
	// Ensure open_ip chains exist
	if err := self.ensureChains(); err != nil {
		return err
	}

	// Add jump rules at the top of internet table chains
	if err := self.addJumpRules(); err != nil {
		return err
	}

	// Check if rule already exists
	exists, err := self.ruleExists(params.DestinationIp, params.MacAddr)
	if err != nil {
		return fmt.Errorf("failed to check if rule exists: %v", err)
	}

	// Create cache key for tracking
	cacheKey := fmt.Sprintf("%s:%s", params.DestinationIp, params.MacAddr)

	if exists {
		// Rule already exists - cancel any existing timer and reschedule if needed
		self.timersMutex.Lock()
		if existingTimer, ok := self.activeTimers[cacheKey]; ok {
			existingTimer.Stop()
			delete(self.activeTimers, cacheKey)
		}
		self.timersMutex.Unlock()

		// Schedule new removal timer if timeout is specified
		if params.TimeoutSecs > 0 {
			self.scheduleRuleRemoval(params.DestinationIp, params.MacAddr, params.TimeoutSecs)
		}
		return nil
	}

	// Determine IP version for destination
	destIpVersion := "ip"
	parsedDestIP := net.ParseIP(params.DestinationIp)
	if parsedDestIP == nil {
		return fmt.Errorf("invalid destination IP address: %s", params.DestinationIp)
	}
	if parsedDestIP.To4() == nil {
		destIpVersion = "ip6"
	}

	// Determine IP version for client
	clientIpVersion := "ip"
	parsedClientIP := net.ParseIP(params.IpAddr)
	if parsedClientIP == nil {
		return fmt.Errorf("invalid client IP address: %s", params.IpAddr)
	}
	if parsedClientIP.To4() == nil {
		clientIpVersion = "ip6"
	}

	// Build nftables rules for bidirectional traffic
	// Outgoing: Client MAC → Destination IP (all ports)
	preroutingOutgoingCmd := fmt.Sprintf("nft add rule inet internet open_ip_prerouting ether saddr %s %s daddr %s counter accept", params.MacAddr, destIpVersion, params.DestinationIp)
	forwardOutgoingCmd := fmt.Sprintf("nft add rule inet internet open_ip_forward ether saddr %s %s daddr %s counter accept", params.MacAddr, destIpVersion, params.DestinationIp)

	// Return: Destination IP → Client IP (all ports)
	preroutingReturnCmd := fmt.Sprintf("nft add rule inet internet open_ip_prerouting %s saddr %s %s daddr %s counter accept", destIpVersion, params.DestinationIp, clientIpVersion, params.IpAddr)
	forwardReturnCmd := fmt.Sprintf("nft add rule inet internet open_ip_forward %s saddr %s %s daddr %s counter accept", destIpVersion, params.DestinationIp, clientIpVersion, params.IpAddr)

	// Add prerouting rule for outgoing traffic (client → destination)
	if err := shell.Exec(preroutingOutgoingCmd, nil); err != nil {
		return fmt.Errorf("failed to add prerouting outgoing rule for %s → %s: %v", params.MacAddr, params.DestinationIp, err)
	}

	// Add forwarding rule for outgoing traffic (client → destination)
	if err := shell.Exec(forwardOutgoingCmd, nil); err != nil {
		return fmt.Errorf("failed to add forward outgoing rule for %s → %s: %v", params.MacAddr, params.DestinationIp, err)
	}

	// Add prerouting rule for return traffic (destination → client)
	if err := shell.Exec(preroutingReturnCmd, nil); err != nil {
		return fmt.Errorf("failed to add prerouting return rule for %s → %s: %v", params.DestinationIp, params.IpAddr, err)
	}

	// Add forwarding rule for return traffic (destination → client)
	if err := shell.Exec(forwardReturnCmd, nil); err != nil {
		return fmt.Errorf("failed to add forward return rule for %s → %s: %v", params.DestinationIp, params.IpAddr, err)
	}

	// Schedule automatic removal if timeout is specified
	if params.TimeoutSecs > 0 {
		self.scheduleRuleRemoval(params.DestinationIp, params.MacAddr, params.TimeoutSecs)
	}

	return nil
}

// scheduleRuleRemoval schedules automatic removal of a firewall rule after the specified timeout
func (self *FirewallApi) scheduleRuleRemoval(destinationIp string, macAddr string, timeoutSecs int) {
	timerKey := fmt.Sprintf("%s:%s", destinationIp, macAddr)

	timer := time.AfterFunc(time.Duration(timeoutSecs)*time.Second, func() {
		// Remove timer from tracking map
		self.timersMutex.Lock()
		delete(self.activeTimers, timerKey)
		self.timersMutex.Unlock()

		// Remove the firewall rule
		err := self.CloseIpForClientDevice(sdkapi.CloseIpForClientDeviceParams{
			DestinationIp: destinationIp,
			MacAddr:       macAddr,
		})
		if err != nil {
			// Log error but don't panic - this is a background operation
			fmt.Printf("Warning: Failed to auto-remove firewall rule for %s (MAC: %s): %v\n", destinationIp, macAddr, err)
		}
	})

	// Store timer in tracking map
	self.timersMutex.Lock()
	self.activeTimers[timerKey] = timer
	self.timersMutex.Unlock()
}

// CloseIpForClientDevice removes firewall access for a specific client device to a destination IP
func (self *FirewallApi) CloseIpForClientDevice(params sdkapi.CloseIpForClientDeviceParams) error {
	// Cancel any active timer for this rule
	cacheKey := fmt.Sprintf("%s:%s", params.DestinationIp, params.MacAddr)
	self.timersMutex.Lock()
	if timer, ok := self.activeTimers[cacheKey]; ok {
		timer.Stop()
		delete(self.activeTimers, cacheKey)
	}
	self.timersMutex.Unlock()

	// Remove the firewall rules for this destination IP
	err := self.removeRulesForDestIP(params.DestinationIp, params.MacAddr)

	return err
}

// removeRulesForDestIP removes firewall rules for a specific destination IP and MAC address
func (self *FirewallApi) removeRulesForDestIP(destinationIp string, macAddr string) error {
	// Determine IP version for destination
	destIpVersion := "ip"
	parsedDestIP := net.ParseIP(destinationIp)
	if parsedDestIP == nil {
		return fmt.Errorf("invalid destination IP address: %s", destinationIp)
	}
	if parsedDestIP.To4() == nil {
		destIpVersion = "ip6"
	}

	// Get all rules and find matching ones to delete
	var out bytes.Buffer
	err := shell.ExecOutput("nft -j list table inet internet", &out)
	if err != nil {
		return fmt.Errorf("failed to list nftables: %v", err)
	}

	var nftOutput nftablesOutput
	if err := json.Unmarshal(out.Bytes(), &nftOutput); err != nil {
		return fmt.Errorf("failed to parse nftables JSON: %v", err)
	}

	// Track if we found and deleted any rules
	deletedAny := false

	// Process each item in the nftables array
	for _, item := range nftOutput.Nftables {
		var rule nftRule
		if err := json.Unmarshal(item, &rule); err != nil {
			continue // Skip non-rule items
		}

		// Only process rules from open_ip chains
		if rule.Rule.Chain != "open_ip_prerouting" && rule.Rule.Chain != "open_ip_forward" {
			continue
		}

		// Check if this rule matches our destination IP and MAC address
		hasDestIPMatch := false
		hasMacMatch := false

		for _, exprRaw := range rule.Rule.Expr {
			var match nftMatch
			if err := json.Unmarshal(exprRaw, &match); err != nil {
				continue
			}

			if match.Match.Op == "" {
				continue
			}

			var payload nftPayload
			if err := json.Unmarshal(match.Match.Left, &payload); err != nil {
				continue
			}

			// Check for destination IP match (daddr or saddr)
			if payload.Payload.Protocol == destIpVersion && (payload.Payload.Field == "daddr" || payload.Payload.Field == "saddr") {
				if ipStr, ok := match.Match.Right.(string); ok && ipStr == destinationIp {
					hasDestIPMatch = true
				}
			}

			// Check for MAC address match (ether saddr)
			if payload.Payload.Protocol == "ether" && payload.Payload.Field == "saddr" {
				if macStr, ok := match.Match.Right.(string); ok && macStr == macAddr {
					hasMacMatch = true
				}
			}
		}

		// Delete rule if it matches destination IP (for return traffic) or MAC + destination IP (for outgoing)
		// We match rules that have either:
		// 1. MAC address match AND dest IP match (outgoing traffic rules)
		// 2. Destination IP as source (return traffic rules - no MAC)
		shouldDelete := hasDestIPMatch && (hasMacMatch || !hasMacMatch)

		if shouldDelete {
			cmd := fmt.Sprintf("nft delete rule inet internet %s handle %d", rule.Rule.Chain, rule.Rule.Handle)
			if err := shell.Exec(cmd, nil); err != nil {
				return fmt.Errorf("failed to delete rule for %s (MAC: %s): %v", destinationIp, macAddr, err)
			}
			deletedAny = true
		}
	}

	if !deletedAny {
		// Not an error - rule might have already been removed
		return nil
	}

	return nil
}

// ensureChains creates open_ip chains within the internet table if they don't exist
func (self *FirewallApi) ensureChains() error {
	// Check if open_ip_prerouting chain exists
	var out bytes.Buffer
	err := shell.ExecOutput("nft -a list chain inet internet open_ip_prerouting 2>&1", &out)
	chainExists := err == nil && !strings.Contains(out.String(), "No such file or directory")

	if !chainExists {
		if err := shell.Exec("nft add chain inet internet open_ip_prerouting", nil); err != nil {
			return fmt.Errorf("failed to create open_ip_prerouting chain: %v", err)
		}
	}

	// Check if open_ip_forward chain exists
	out.Reset()
	err = shell.ExecOutput("nft -a list chain inet internet open_ip_forward 2>&1", &out)
	chainExists = err == nil && !strings.Contains(out.String(), "No such file or directory")

	if !chainExists {
		if err := shell.Exec("nft add chain inet internet open_ip_forward", nil); err != nil {
			return fmt.Errorf("failed to create open_ip_forward chain: %v", err)
		}
	}

	return nil
}

// addJumpRules adds jump rules at the top of internet table chains
func (self *FirewallApi) addJumpRules() error {
	// Check if jump rule already exists in prerouting
	var out bytes.Buffer
	err := shell.ExecOutput("nft -a list chain inet internet prerouting", &out)
	if err != nil {
		// If chain doesn't exist, try to add the jump rule
		if strings.Contains(out.String(), "No such file or directory") {
			if err := shell.Exec("nft insert rule inet internet prerouting counter jump open_ip_prerouting", nil); err != nil {
				return fmt.Errorf("failed to add prerouting jump rule: %v", err)
			}
		} else {
			return fmt.Errorf("failed to list prerouting chain: %v", err)
		}
	} else if !strings.Contains(out.String(), "jump open_ip_prerouting") {
		if err := shell.Exec("nft insert rule inet internet prerouting counter jump open_ip_prerouting", nil); err != nil {
			return fmt.Errorf("failed to add prerouting jump rule: %v", err)
		}
	}

	// Check if jump rule already exists in forward
	out.Reset()
	err = shell.ExecOutput("nft -a list chain inet internet forward", &out)
	if err != nil {
		// If chain doesn't exist, try to add the jump rule
		if strings.Contains(out.String(), "No such file or directory") {
			if err := shell.Exec("nft insert rule inet internet forward counter jump open_ip_forward", nil); err != nil {
				return fmt.Errorf("failed to add forward jump rule: %v", err)
			}
		} else {
			return fmt.Errorf("failed to list forward chain: %v", err)
		}
	} else if !strings.Contains(out.String(), "jump open_ip_forward") {
		if err := shell.Exec("nft insert rule inet internet forward counter jump open_ip_forward", nil); err != nil {
			return fmt.Errorf("failed to add forward jump rule: %v", err)
		}
	}

	return nil
}

// ruleExists checks if a rule for the given destination IP and MAC address already exists
func (self *FirewallApi) ruleExists(destinationIp string, macAddr string) (bool, error) {
	var out bytes.Buffer
	err := shell.ExecOutput("nft -j list table inet internet", &out)
	if err != nil {
		return false, fmt.Errorf("failed to list nftables: %v", err)
	}

	var nftOutput nftablesOutput
	if err := json.Unmarshal(out.Bytes(), &nftOutput); err != nil {
		return false, fmt.Errorf("failed to parse nftables JSON: %v", err)
	}

	// Determine IP version
	ipVersion := "ip"
	parsedIP := net.ParseIP(destinationIp)
	if parsedIP != nil && parsedIP.To4() == nil {
		ipVersion = "ip6"
	}

	// We need to find at least one outgoing rule (MAC + dest IP as daddr)
	// The presence of outgoing rules indicates the full rule set exists
	// because OpenIpForClientDevice() adds all 4 rules atomically
	for _, item := range nftOutput.Nftables {
		var rule nftRule
		if err := json.Unmarshal(item, &rule); err != nil {
			continue // Skip non-rule items
		}

		// Only process rules from open_ip chains
		if rule.Rule.Chain != "open_ip_prerouting" && rule.Rule.Chain != "open_ip_forward" {
			continue
		}

		// Check if this is an outgoing rule (MAC + dest IP as daddr)
		hasDestIPAsDaddr := false
		hasMacMatch := false

		for _, exprRaw := range rule.Rule.Expr {
			var match nftMatch
			if err := json.Unmarshal(exprRaw, &match); err != nil {
				continue
			}

			if match.Match.Op == "" {
				continue
			}

			var payload nftPayload
			if err := json.Unmarshal(match.Match.Left, &payload); err != nil {
				continue
			}

			// Check for destination IP as daddr (outgoing traffic)
			if payload.Payload.Protocol == ipVersion && payload.Payload.Field == "daddr" {
				if ipStr, ok := match.Match.Right.(string); ok && ipStr == destinationIp {
					hasDestIPAsDaddr = true
				}
			}

			// Check for MAC address match (ether saddr)
			if payload.Payload.Protocol == "ether" && payload.Payload.Field == "saddr" {
				if macStr, ok := match.Match.Right.(string); ok && macStr == macAddr {
					hasMacMatch = true
				}
			}
		}

		// If we find an outgoing rule (MAC + dest IP as daddr), the rule set exists
		if hasDestIPAsDaddr && hasMacMatch {
			return true, nil
		}
	}

	return false, nil
}

// RemoveJumpRulesAndChains removes jump rules and cleans up open_ip chains
func (self *FirewallApi) RemoveJumpRulesAndChains() error {
	// Remove jump rule from prerouting
	deleteCmd := "nft delete rule inet internet prerouting handle $(nft -a list chain inet internet prerouting | grep 'jump open_ip_prerouting' | awk '{print $NF}')"
	var out bytes.Buffer
	err := shell.ExecOutput(deleteCmd, &out)
	if err != nil && !strings.Contains(out.String(), "No such file or directory") {
		return fmt.Errorf("failed to remove prerouting jump rule: %v (%s)", err, out.String())
	}

	// Remove jump rule from forward
	deleteCmd = "nft delete rule inet internet forward handle $(nft -a list chain inet internet forward | grep 'jump open_ip_forward' | awk '{print $NF}')"
	out.Reset()
	err = shell.ExecOutput(deleteCmd, &out)
	if err != nil && !strings.Contains(out.String(), "No such file or directory") {
		return fmt.Errorf("failed to remove forward jump rule: %v (%s)", err, out.String())
	}

	// Flush and delete open_ip chains
	cmds := []string{
		"nft flush chain inet internet open_ip_prerouting",
		"nft flush chain inet internet open_ip_forward",
		"nft delete chain inet internet open_ip_prerouting",
		"nft delete chain inet internet open_ip_forward",
	}

	for _, cmd := range cmds {
		out.Reset()
		err := shell.ExecOutput(cmd, &out)
		if err != nil && !strings.Contains(out.String(), "No such file or directory") {
			// Log warning but don't fail
			fmt.Printf("Warning: Failed to execute cleanup command '%s': %v (%s)\n", cmd, err, out.String())
		}
	}

	return nil
}

var _ sdkapi.IFirewallAPI = (*FirewallApi)(nil)
