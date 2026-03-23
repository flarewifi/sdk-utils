package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	jobque "core/utils/job-que"
	"core/utils/shell"
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	// DstIpGroupMaxAge is the maximum age for IPs in a destination group before they're flushed
	DstIpGroupMaxAge = 12 * time.Hour
	// dnsResolveCacheTTL is how long to keep cached DNS resolution results fresh
	dnsResolveCacheTTL = 30 * time.Minute
	// dnsResolveCacheStaleMax is the max age before refusing to use stale cached DNS data
	dnsResolveCacheStaleMax = 24 * time.Hour
)

// dnsResolveEntry stores cached DNS resolution results with a timestamp
type dnsResolveEntry struct {
	ips       []string
	timestamp time.Time
}

// servicePortDefinition stores the definition of a service port (protocols, port, destination IPs)
type servicePortDefinition struct {
	Protocols []string
	Port      int
	DstIPv4   []string
	DstIPv6   []string
}

func NewFirewallApi(api *PluginApi) {
	firewallApi := &FirewallApi{
		activeTimers:        make(map[string]*time.Timer),
		firewallMutex:       &sync.RWMutex{},
		firewallQue:         jobque.NewJobQueue[any](),
		createdGroups:       make(map[string]bool),
		groupIPs:            make(map[string]map[string]time.Time),
		dnsResolveCache:     make(map[string]dnsResolveEntry),
		dnsCacheMutex:       &sync.RWMutex{},
		createdServicePorts: make(map[string]bool),
		servicePortDefs:     make(map[string]servicePortDefinition),
	}
	api.FirewallAPI = firewallApi
}

type FirewallApi struct {
	activeTimers        map[string]*time.Timer           // Track active removal timers by key (group/service port specific)
	firewallMutex       *sync.RWMutex                    // Protect concurrent access to maps
	firewallQue         *jobque.JobQueue[any]            // Serialize firewall operations to prevent race conditions
	createdGroups       map[string]bool                  // Track created destination IP groups by slugified name
	groupIPs            map[string]map[string]time.Time  // Track IPs per group with timestamp when added (slugName -> IP -> addedAt)
	dnsResolveCache     map[string]dnsResolveEntry       // Cache DNS resolution results by hostname
	dnsCacheMutex       *sync.RWMutex                    // Protect concurrent access to dnsResolveCache
	createdServicePorts map[string]bool                  // Track created service ports by slugified name
	servicePortDefs     map[string]servicePortDefinition // Store service port definitions by slugified name
}

// nftRuleListResult represents the JSON output of "nft -j -a list chain"
type nftRuleListResult struct {
	Nftables []nftRuleEntry `json:"nftables"`
}

type nftRuleEntry struct {
	Rule *nftRule `json:"rule,omitempty"`
}

type nftRule struct {
	Family string        `json:"family"`
	Table  string        `json:"table"`
	Chain  string        `json:"chain"`
	Handle int           `json:"handle"`
	Expr   []interface{} `json:"expr"` // Mixed types, we'll parse manually
}

// ResolveHostnameToIps is implemented in firewall-api-resolve.go and firewall-api-resolve_dev.go
// with different behavior for dev and production builds

// CreateDstIpGroup creates a named group of destination IP addresses with dedicated nftables infrastructure.
// The group name is slugified for safe use in nftables identifiers.
// Returns error if group already exists or if any IP address is invalid.
func (self *FirewallApi) CreateDstIpGroup(name string, ips ...string) error {
	// Slugify the group name for safe nftables identifiers
	slugName := sdkutils.Slugify(name, "_")
	if slugName == "" {
		return fmt.Errorf("invalid group name: %s (must contain alphanumeric characters)", name)
	}

	// Separate IPs by version (validates all IPs)
	separated, err := sdkutils.SeparateIPsByVersion(ips)
	if err != nil {
		return err
	}

	contextInfo := fmt.Sprintf("GroupName=%s (slug=%s)", name, slugName)

	_, err = self.firewallQue.ExecWithTimeout(
		10*time.Second,
		"Create Destination IP Group",
		contextInfo,
		func() (any, error) {
			return nil, self.doCreateDstIpGroup(slugName, separated)
		},
	)
	return err
}

// doCreateDstIpGroup is the internal implementation of CreateDstIpGroup
func (self *FirewallApi) doCreateDstIpGroup(slugName string, ips sdkutils.SeparatedIPs) error {
	// Check if group already exists
	self.firewallMutex.RLock()
	if self.createdGroups[slugName] {
		self.firewallMutex.RUnlock()
		return fmt.Errorf("destination IP group already exists: %s", slugName)
	}
	self.firewallMutex.RUnlock()

	// Define nftables resource names
	setV4 := fmt.Sprintf("dst_grp_%s_v4", slugName)
	setV6 := fmt.Sprintf("dst_grp_%s_v6", slugName)
	setMacs := fmt.Sprintf("dst_grp_%s_macs", slugName)
	setClientIpsV4 := fmt.Sprintf("dst_grp_%s_client_ips_v4", slugName)
	setClientIpsV6 := fmt.Sprintf("dst_grp_%s_client_ips_v6", slugName)
	chainPrerouting := fmt.Sprintf("dst_grp_%s_prerouting", slugName)
	chainForward := fmt.Sprintf("dst_grp_%s_forward", slugName)

	// Build nft batch script for atomic execution
	var batch strings.Builder
	batch.WriteString("table inet internet {\n")

	// Create sets for destination IPs
	batch.WriteString(fmt.Sprintf("\tset %s {\n\t\ttype ipv4_addr\n\t}\n", setV4))
	batch.WriteString(fmt.Sprintf("\tset %s {\n\t\ttype ipv6_addr\n\t}\n", setV6))

	// Create sets for client MACs and IPs (for return traffic)
	batch.WriteString(fmt.Sprintf("\tset %s {\n\t\ttype ether_addr\n\t}\n", setMacs))
	batch.WriteString(fmt.Sprintf("\tset %s {\n\t\ttype ipv4_addr\n\t}\n", setClientIpsV4))
	batch.WriteString(fmt.Sprintf("\tset %s {\n\t\ttype ipv6_addr\n\t}\n", setClientIpsV6))

	// Create chains with rules
	// Prerouting chain
	batch.WriteString(fmt.Sprintf("\tchain %s {\n", chainPrerouting))
	batch.WriteString(fmt.Sprintf("\t\tether saddr @%s ip daddr @%s counter accept\n", setMacs, setV4))
	batch.WriteString(fmt.Sprintf("\t\tether saddr @%s ip6 daddr @%s counter accept\n", setMacs, setV6))
	batch.WriteString(fmt.Sprintf("\t\tip saddr @%s ip daddr @%s counter accept\n", setV4, setClientIpsV4))
	batch.WriteString(fmt.Sprintf("\t\tip6 saddr @%s ip6 daddr @%s counter accept\n", setV6, setClientIpsV6))
	batch.WriteString("\t}\n")

	// Forward chain
	batch.WriteString(fmt.Sprintf("\tchain %s {\n", chainForward))
	batch.WriteString(fmt.Sprintf("\t\tether saddr @%s ip daddr @%s counter accept\n", setMacs, setV4))
	batch.WriteString(fmt.Sprintf("\t\tether saddr @%s ip6 daddr @%s counter accept\n", setMacs, setV6))
	batch.WriteString(fmt.Sprintf("\t\tip saddr @%s ip daddr @%s counter accept\n", setV4, setClientIpsV4))
	batch.WriteString(fmt.Sprintf("\t\tip6 saddr @%s ip6 daddr @%s counter accept\n", setV6, setClientIpsV6))
	batch.WriteString("\t}\n")

	batch.WriteString("}\n")

	// Add destination IPs to sets if provided
	if len(ips.IPv4) > 0 {
		ipList := strings.Join(ips.IPv4, ", ")
		batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setV4, ipList))
	}
	if len(ips.IPv6) > 0 {
		ipList := strings.Join(ips.IPv6, ", ")
		batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setV6, ipList))
	}

	// Add jump rules from main chains to group chains.
	// Prerouting: insert (must run before captive portal DNAT rules).
	// Forward: append (runs after MAC/IP verdict maps so connected clients have lowest latency).
	batch.WriteString(fmt.Sprintf("insert rule inet internet prerouting counter jump %s\n", chainPrerouting))
	batch.WriteString(fmt.Sprintf("add rule inet internet forward counter jump %s\n", chainForward))

	// Prepare new IP map with current timestamps (before executing nftables)
	now := time.Now()
	newIPMap := make(map[string]time.Time)
	for _, ip := range ips.IPv4 {
		newIPMap[ip] = now
	}
	for _, ip := range ips.IPv6 {
		newIPMap[ip] = now
	}

	// Execute batch command using nft -f - with heredoc for safe shell escaping
	cmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
	if err := shell.Exec(cmd, nil); err != nil {
		return fmt.Errorf("failed to create destination IP group: %v", err)
	}

	// Mark group as created and track IPs (only after nftables success)
	self.firewallMutex.Lock()
	self.createdGroups[slugName] = true
	self.groupIPs[slugName] = newIPMap
	self.firewallMutex.Unlock()

	return nil
}

// AllowClientToDstIpGroup allows a specific client device to access all IPs in a named destination IP group.
// The client's MAC and IP are added to the group's client sets, enabling access through the group's firewall rules.
// If timeoutSecs > 0, the client is automatically removed after the specified duration.
func (self *FirewallApi) AllowClientToDstIpGroup(clnt sdkapi.DstIpGroupClient, groupName string, timeoutSecs int) error {
	// Slugify the group name to match how it was created
	slugName := sdkutils.Slugify(groupName, "_")
	if slugName == "" {
		return fmt.Errorf("invalid group name: %s (must contain alphanumeric characters)", groupName)
	}

	contextInfo := fmt.Sprintf("GroupName=%s, ClientMAC=%s, ClientIP=%s", groupName, clnt.MacAddr, clnt.IpAddr)

	_, err := self.firewallQue.ExecWithTimeout(
		5*time.Second,
		"Allow Client to Destination IP Group",
		contextInfo,
		func() (any, error) {
			return nil, self.doAllowClientToDstIpGroup(clnt, slugName, timeoutSecs)
		},
	)
	return err
}

// doAllowClientToDstIpGroup is the internal implementation of AllowClientToDstIpGroup
func (self *FirewallApi) doAllowClientToDstIpGroup(clnt sdkapi.DstIpGroupClient, slugName string, timeoutSecs int) error {
	// Check if group exists
	self.firewallMutex.RLock()
	if !self.createdGroups[slugName] {
		self.firewallMutex.RUnlock()
		return fmt.Errorf("destination IP group does not exist: %s", slugName)
	}
	self.firewallMutex.RUnlock()

	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(clnt.MacAddr)
	if err != nil {
		return fmt.Errorf("MAC validation failed: %v", err)
	}
	clnt.MacAddr = normalizedMAC

	// Resolve effective IPv4/IPv6 addresses.
	// Support both new-style (Ipv4Addr/Ipv6Addr) and legacy (IpAddr) fields.
	ipv4, ipv6 := clnt.Ipv4Addr, clnt.Ipv6Addr
	if ipv4 == "" && ipv6 == "" && clnt.IpAddr != "" {
		ver, err := sdkutils.GetIPVersion(clnt.IpAddr)
		if err != nil {
			return fmt.Errorf("client IP validation failed: %v", err)
		}
		if ver == "ip" {
			ipv4 = clnt.IpAddr
		} else {
			ipv6 = clnt.IpAddr
		}
	}
	if ipv4 == "" && ipv6 == "" {
		return fmt.Errorf("at least one of Ipv4Addr or Ipv6Addr must be set for client %s", clnt.MacAddr)
	}

	// Validate each IP address that is present
	if ipv4 != "" {
		if _, err := sdkutils.ValidateIPAddress(ipv4); err != nil {
			return fmt.Errorf("client IPv4 validation failed: %v", err)
		}
	}
	if ipv6 != "" {
		if _, err := sdkutils.ValidateIPAddress(ipv6); err != nil {
			return fmt.Errorf("client IPv6 validation failed: %v", err)
		}
	}

	// Define nftables set names
	setMacs := fmt.Sprintf("dst_grp_%s_macs", slugName)
	setClientIpsV4 := fmt.Sprintf("dst_grp_%s_client_ips_v4", slugName)
	setClientIpsV6 := fmt.Sprintf("dst_grp_%s_client_ips_v6", slugName)

	// Cancel any existing timer for this client in this group
	cacheKey := fmt.Sprintf("grp:%s:%s", slugName, clnt.MacAddr)
	self.firewallMutex.Lock()
	if existingTimer, ok := self.activeTimers[cacheKey]; ok {
		existingTimer.Stop()
		delete(self.activeTimers, cacheKey)
	}
	self.firewallMutex.Unlock()

	// Build nft batch script to add client MAC and both IP addresses (where present)
	var batch strings.Builder
	batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setMacs, clnt.MacAddr))
	if ipv4 != "" {
		batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setClientIpsV4, ipv4))
	}
	if ipv6 != "" {
		batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setClientIpsV6, ipv6))
	}

	// Execute batch command using heredoc for safe shell escaping
	nftCmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
	if err := shell.Exec(nftCmd, nil); err != nil {
		return fmt.Errorf("failed to add client to destination IP group: %v", err)
	}

	// Schedule automatic removal if timeout is specified
	if timeoutSecs > 0 {
		self.scheduleGroupClientRemoval(slugName, clnt, timeoutSecs)
	}

	return nil
}

// scheduleGroupClientRemoval schedules automatic removal of a client from a destination IP group
func (self *FirewallApi) scheduleGroupClientRemoval(slugName string, clnt sdkapi.DstIpGroupClient, timeoutSecs int) {
	cacheKey := fmt.Sprintf("grp:%s:%s", slugName, clnt.MacAddr)

	timer := time.AfterFunc(time.Duration(timeoutSecs)*time.Second, func() {
		// Remove timer from tracking map
		self.firewallMutex.Lock()
		delete(self.activeTimers, cacheKey)
		self.firewallMutex.Unlock()

		// Remove the client from the group
		err := self.RemoveClientFromDstIpGroup(clnt, slugName)
		if err != nil {
			// Log error but don't panic - this is a background operation
			fmt.Printf("Warning: Failed to auto-remove client %s from group %s: %v\n", clnt.MacAddr, slugName, err)
		}
	})

	// Store timer in tracking map
	self.firewallMutex.Lock()
	self.activeTimers[cacheKey] = timer
	self.firewallMutex.Unlock()
}

// RemoveClientFromDstIpGroup removes access for a specific client device from a named destination IP group.
// The client's MAC and IP are removed from the group's client sets.
func (self *FirewallApi) RemoveClientFromDstIpGroup(clnt sdkapi.DstIpGroupClient, groupName string) error {
	// Slugify the group name to match how it was created
	slugName := sdkutils.Slugify(groupName, "_")
	if slugName == "" {
		return fmt.Errorf("invalid group name: %s (must contain alphanumeric characters)", groupName)
	}

	contextInfo := fmt.Sprintf("GroupName=%s, ClientMAC=%s, ClientIP=%s", groupName, clnt.MacAddr, clnt.IpAddr)

	_, err := self.firewallQue.ExecWithTimeout(
		5*time.Second,
		"Remove Client from Destination IP Group",
		contextInfo,
		func() (any, error) {
			return nil, self.doRemoveClientFromDstIpGroup(clnt, slugName)
		},
	)
	return err
}

// doRemoveClientFromDstIpGroup is the internal implementation of RemoveClientFromDstIpGroup
func (self *FirewallApi) doRemoveClientFromDstIpGroup(clnt sdkapi.DstIpGroupClient, slugName string) error {
	// Check if group exists
	self.firewallMutex.RLock()
	if !self.createdGroups[slugName] {
		self.firewallMutex.RUnlock()
		return fmt.Errorf("destination IP group does not exist: %s", slugName)
	}
	self.firewallMutex.RUnlock()

	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(clnt.MacAddr)
	if err != nil {
		return fmt.Errorf("MAC validation failed: %v", err)
	}
	clnt.MacAddr = normalizedMAC

	// Resolve effective IPv4/IPv6 addresses (same logic as doAllowClientToDstIpGroup)
	ipv4, ipv6 := clnt.Ipv4Addr, clnt.Ipv6Addr
	if ipv4 == "" && ipv6 == "" && clnt.IpAddr != "" {
		ver, err := sdkutils.GetIPVersion(clnt.IpAddr)
		if err == nil {
			if ver == "ip" {
				ipv4 = clnt.IpAddr
			} else {
				ipv6 = clnt.IpAddr
			}
		}
	}

	// Define nftables set names
	setMacs := fmt.Sprintf("dst_grp_%s_macs", slugName)
	setClientIpsV4 := fmt.Sprintf("dst_grp_%s_client_ips_v4", slugName)
	setClientIpsV6 := fmt.Sprintf("dst_grp_%s_client_ips_v6", slugName)

	// Cancel any active timer for this client in this group
	cacheKey := fmt.Sprintf("grp:%s:%s", slugName, clnt.MacAddr)
	self.firewallMutex.Lock()
	if timer, ok := self.activeTimers[cacheKey]; ok {
		timer.Stop()
		delete(self.activeTimers, cacheKey)
	}
	self.firewallMutex.Unlock()

	// Remove MAC and both IP addresses (best-effort — elements may already be absent).
	// Each delete is run as a separate shell command so that a missing element in one
	// set does not prevent cleanup of the others.
	shell.Exec(fmt.Sprintf("nft delete element inet internet %s '{ %s }' 2>/dev/null || true", setMacs, clnt.MacAddr), nil)
	if ipv4 != "" {
		shell.Exec(fmt.Sprintf("nft delete element inet internet %s '{ %s }' 2>/dev/null || true", setClientIpsV4, ipv4), nil)
	}
	if ipv6 != "" {
		shell.Exec(fmt.Sprintf("nft delete element inet internet %s '{ %s }' 2>/dev/null || true", setClientIpsV6, ipv6), nil)
	}

	return nil
}

// AddIpsToDstIpGroup adds IP addresses to an existing named destination IP group.
// The new IPs are merged with existing IPs in the group.
// Returns an error if the group does not exist or if any IP address is invalid.
func (self *FirewallApi) AddIpsToDstIpGroup(name string, ips ...string) error {
	// Slugify the group name to match how it was created
	slugName := sdkutils.Slugify(name, "_")
	if slugName == "" {
		return fmt.Errorf("invalid group name: %s (must contain alphanumeric characters)", name)
	}

	// Separate IPs by version (validates all IPs)
	separated, err := sdkutils.SeparateIPsByVersion(ips)
	if err != nil {
		return err
	}

	contextInfo := fmt.Sprintf("GroupName=%s (slug=%s)", name, slugName)

	_, err = self.firewallQue.ExecWithTimeout(
		5*time.Second,
		"Add IPs to Destination IP Group",
		contextInfo,
		func() (any, error) {
			return nil, self.doAddIpsToDstIpGroup(slugName, separated)
		},
	)
	return err
}

// doAddIpsToDstIpGroup is the internal implementation of AddIpsToDstIpGroup
func (self *FirewallApi) doAddIpsToDstIpGroup(slugName string, ips sdkutils.SeparatedIPs) error {
	now := time.Now()
	cutoff := now.Add(-DstIpGroupMaxAge)

	// Check if group exists and get existing IPs
	self.firewallMutex.RLock()
	if !self.createdGroups[slugName] {
		self.firewallMutex.RUnlock()
		return fmt.Errorf("destination IP group does not exist: %s", slugName)
	}
	existingIPs := self.groupIPs[slugName]
	self.firewallMutex.RUnlock()

	// Nothing to add
	if len(ips.IPv4) == 0 && len(ips.IPv6) == 0 {
		return nil
	}

	// Check if any existing IPs are stale (older than 12 hours)
	hasStaleIPs := false
	for _, addedAt := range existingIPs {
		if addedAt.Before(cutoff) {
			hasStaleIPs = true
			break
		}
	}

	// Define nftables set names
	setV4 := fmt.Sprintf("dst_grp_%s_v4", slugName)
	setV6 := fmt.Sprintf("dst_grp_%s_v6", slugName)

	var batch strings.Builder
	var newIPMap map[string]time.Time

	if hasStaleIPs {
		// FLUSH mode: flush sets, add all current IPs, reset tracking
		batch.WriteString(fmt.Sprintf("flush set inet internet %s\n", setV4))
		batch.WriteString(fmt.Sprintf("flush set inet internet %s\n", setV6))

		if len(ips.IPv4) > 0 {
			ipList := strings.Join(ips.IPv4, ", ")
			batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setV4, ipList))
		}
		if len(ips.IPv6) > 0 {
			ipList := strings.Join(ips.IPv6, ", ")
			batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setV6, ipList))
		}

		// Prepare new map with all IPs at current timestamp
		newIPMap = make(map[string]time.Time)
		for _, ip := range ips.IPv4 {
			newIPMap[ip] = now
		}
		for _, ip := range ips.IPv6 {
			newIPMap[ip] = now
		}
	} else {
		// ADD mode: filter existing, add only new IPs
		var newIPv4, newIPv6 []string
		for _, ip := range ips.IPv4 {
			if _, exists := existingIPs[ip]; !exists {
				newIPv4 = append(newIPv4, ip)
			}
		}
		for _, ip := range ips.IPv6 {
			if _, exists := existingIPs[ip]; !exists {
				newIPv6 = append(newIPv6, ip)
			}
		}

		// Nothing new to add
		if len(newIPv4) == 0 && len(newIPv6) == 0 {
			return nil
		}

		if len(newIPv4) > 0 {
			ipList := strings.Join(newIPv4, ", ")
			batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setV4, ipList))
		}
		if len(newIPv6) > 0 {
			ipList := strings.Join(newIPv6, ", ")
			batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setV6, ipList))
		}

		// Prepare updated map: copy existing + add new
		newIPMap = make(map[string]time.Time, len(existingIPs)+len(newIPv4)+len(newIPv6))
		for ip, ts := range existingIPs {
			newIPMap[ip] = ts
		}
		for _, ip := range newIPv4 {
			newIPMap[ip] = now
		}
		for _, ip := range newIPv6 {
			newIPMap[ip] = now
		}
	}

	// Execute batch command using heredoc for safe shell escaping
	cmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
	if err := shell.Exec(cmd, nil); err != nil {
		return fmt.Errorf("failed to add IPs to destination IP group: %v", err)
	}

	// Update in-memory map only after nftables success
	self.firewallMutex.Lock()
	self.groupIPs[slugName] = newIPMap
	self.firewallMutex.Unlock()

	return nil
}

// ChangeDstIpGroup replaces all IP addresses in an existing named destination IP group.
// All existing IPs are removed and replaced with the new set.
// Returns an error if the group does not exist or if any IP address is invalid.
func (self *FirewallApi) ChangeDstIpGroup(name string, ips ...string) error {
	// Slugify the group name to match how it was created
	slugName := sdkutils.Slugify(name, "_")
	if slugName == "" {
		return fmt.Errorf("invalid group name: %s (must contain alphanumeric characters)", name)
	}

	// Separate IPs by version (validates all IPs)
	separated, err := sdkutils.SeparateIPsByVersion(ips)
	if err != nil {
		return err
	}

	contextInfo := fmt.Sprintf("GroupName=%s (slug=%s)", name, slugName)

	_, err = self.firewallQue.ExecWithTimeout(
		5*time.Second,
		"Change Destination IP Group",
		contextInfo,
		func() (any, error) {
			return nil, self.doChangeDstIpGroup(slugName, separated)
		},
	)
	return err
}

// DstIpGroupExists checks if a named destination IP group exists.
// Returns true if the group was created and is tracked, false otherwise.
func (self *FirewallApi) DstIpGroupExists(name string) (bool, error) {
	// Slugify the group name to match how it was created
	slugName := sdkutils.Slugify(name, "_")
	if slugName == "" {
		return false, fmt.Errorf("invalid group name: %s (must contain alphanumeric characters)", name)
	}

	self.firewallMutex.RLock()
	exists := self.createdGroups[slugName]
	self.firewallMutex.RUnlock()

	return exists, nil
}

// DeleteDstIpGroup removes a named destination IP group and all its nftables infrastructure.
// All clients currently allowed access through this group will immediately lose access.
func (self *FirewallApi) DeleteDstIpGroup(name string) error {
	// Slugify the group name to match how it was created
	slugName := sdkutils.Slugify(name, "_")
	if slugName == "" {
		return fmt.Errorf("invalid group name: %s (must contain alphanumeric characters)", name)
	}

	contextInfo := fmt.Sprintf("GroupName=%s (slug=%s)", name, slugName)

	_, err := self.firewallQue.ExecWithTimeout(
		10*time.Second,
		"Delete Destination IP Group",
		contextInfo,
		func() (any, error) {
			return nil, self.doDeleteDstIpGroup(slugName)
		},
	)
	return err
}

// doDeleteDstIpGroup is the internal implementation of DeleteDstIpGroup
func (self *FirewallApi) doDeleteDstIpGroup(slugName string) error {
	// Check if group exists
	self.firewallMutex.RLock()
	if !self.createdGroups[slugName] {
		self.firewallMutex.RUnlock()
		return fmt.Errorf("destination IP group does not exist: %s", slugName)
	}
	self.firewallMutex.RUnlock()

	// Cancel all timers for this group
	self.cancelTimersWithPrefix("grp:" + slugName + ":")

	// Define nftables resource names
	chainPrerouting := fmt.Sprintf("dst_grp_%s_prerouting", slugName)
	chainForward := fmt.Sprintf("dst_grp_%s_forward", slugName)
	setV4 := fmt.Sprintf("dst_grp_%s_v4", slugName)
	setV6 := fmt.Sprintf("dst_grp_%s_v6", slugName)
	setMacs := fmt.Sprintf("dst_grp_%s_macs", slugName)
	setClientIpsV4 := fmt.Sprintf("dst_grp_%s_client_ips_v4", slugName)
	setClientIpsV6 := fmt.Sprintf("dst_grp_%s_client_ips_v6", slugName)

	// Delete jump rules from main chains
	self.deleteJumpRule("prerouting", chainPrerouting)
	self.deleteJumpRule("forward", chainForward)

	// Build batch script to delete chains and sets
	var batch strings.Builder

	// Flush and delete chains
	batch.WriteString(fmt.Sprintf("flush chain inet internet %s 2>/dev/null || true\n", chainPrerouting))
	batch.WriteString(fmt.Sprintf("delete chain inet internet %s 2>/dev/null || true\n", chainPrerouting))
	batch.WriteString(fmt.Sprintf("flush chain inet internet %s 2>/dev/null || true\n", chainForward))
	batch.WriteString(fmt.Sprintf("delete chain inet internet %s 2>/dev/null || true\n", chainForward))

	// Delete sets
	batch.WriteString(fmt.Sprintf("delete set inet internet %s 2>/dev/null || true\n", setV4))
	batch.WriteString(fmt.Sprintf("delete set inet internet %s 2>/dev/null || true\n", setV6))
	batch.WriteString(fmt.Sprintf("delete set inet internet %s 2>/dev/null || true\n", setMacs))
	batch.WriteString(fmt.Sprintf("delete set inet internet %s 2>/dev/null || true\n", setClientIpsV4))
	batch.WriteString(fmt.Sprintf("delete set inet internet %s 2>/dev/null || true\n", setClientIpsV6))

	// Execute batch command
	cmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
	if err := shell.Exec(cmd, nil); err != nil {
		return fmt.Errorf("failed to delete destination IP group infrastructure: %v", err)
	}

	// Remove from tracking maps
	self.firewallMutex.Lock()
	delete(self.createdGroups, slugName)
	delete(self.groupIPs, slugName)
	self.firewallMutex.Unlock()

	return nil
}

// doChangeDstIpGroup is the internal implementation of ChangeDstIpGroup
func (self *FirewallApi) doChangeDstIpGroup(slugName string, ips sdkutils.SeparatedIPs) error {
	// Check if group exists
	self.firewallMutex.RLock()
	if !self.createdGroups[slugName] {
		self.firewallMutex.RUnlock()
		return fmt.Errorf("destination IP group does not exist: %s", slugName)
	}
	self.firewallMutex.RUnlock()

	// Define nftables set names
	setV4 := fmt.Sprintf("dst_grp_%s_v4", slugName)
	setV6 := fmt.Sprintf("dst_grp_%s_v6", slugName)

	// Build nft batch script: flush sets, then add new elements
	var batch strings.Builder
	batch.WriteString(fmt.Sprintf("flush set inet internet %s\n", setV4))
	batch.WriteString(fmt.Sprintf("flush set inet internet %s\n", setV6))

	if len(ips.IPv4) > 0 {
		ipList := strings.Join(ips.IPv4, ", ")
		batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setV4, ipList))
	}
	if len(ips.IPv6) > 0 {
		ipList := strings.Join(ips.IPv6, ", ")
		batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setV6, ipList))
	}

	// Prepare new IP map with current timestamps (before executing nftables)
	now := time.Now()
	newIPMap := make(map[string]time.Time)
	for _, ip := range ips.IPv4 {
		newIPMap[ip] = now
	}
	for _, ip := range ips.IPv6 {
		newIPMap[ip] = now
	}

	// Execute batch command using heredoc for safe shell escaping
	cmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
	if err := shell.Exec(cmd, nil); err != nil {
		return fmt.Errorf("failed to change destination IP group: %v", err)
	}

	// Reset IP tracking with new timestamps (only after nftables success)
	self.firewallMutex.Lock()
	self.groupIPs[slugName] = newIPMap
	self.firewallMutex.Unlock()

	return nil
}

// =============================================================================
// SERVICE PORT METHODS (new pattern - replaces old ServiceDef)
// =============================================================================

// CreateServicePort creates a named service port definition that allows traffic on specified protocol(s) and port.
// The service port uses nftables sets and dedicated chains (similar to DstIpGroup pattern).
func (self *FirewallApi) CreateServicePort(name string, protocols []string, port int, dstIPs ...string) error {
	// Slugify the service port name for safe nftables identifiers
	slugName := sdkutils.Slugify(name, "_")
	if slugName == "" {
		return fmt.Errorf("invalid service port name: %s (must contain alphanumeric characters)", name)
	}

	// Validate protocols and port
	if err := self.validateServicePortParams(protocols, port); err != nil {
		return err
	}

	// Separate IPs by version (validates all IPs)
	var separated sdkutils.SeparatedIPs
	if len(dstIPs) > 0 {
		var err error
		separated, err = sdkutils.SeparateIPsByVersion(dstIPs)
		if err != nil {
			return err
		}
	}

	contextInfo := fmt.Sprintf("ServicePortName=%s (slug=%s), Protocols=%v, Port=%d", name, slugName, protocols, port)

	_, err := self.firewallQue.ExecWithTimeout(
		10*time.Second,
		"Create Service Port",
		contextInfo,
		func() (any, error) {
			return nil, self.doCreateServicePort(slugName, protocols, port, separated)
		},
	)
	return err
}

// doCreateServicePort is the internal implementation of CreateServicePort
func (self *FirewallApi) doCreateServicePort(slugName string, protocols []string, port int, dstIPs sdkutils.SeparatedIPs) error {
	// Check if service port already exists
	self.firewallMutex.RLock()
	if self.createdServicePorts[slugName] {
		self.firewallMutex.RUnlock()
		return fmt.Errorf("service port already exists: %s", slugName)
	}
	self.firewallMutex.RUnlock()

	// Define nftables resource names
	setMacs := fmt.Sprintf("svc_port_%s_macs", slugName)
	setClientIpsV4 := fmt.Sprintf("svc_port_%s_client_ips_v4", slugName)
	setClientIpsV6 := fmt.Sprintf("svc_port_%s_client_ips_v6", slugName)
	setDstV4 := fmt.Sprintf("svc_port_%s_dst_v4", slugName)
	setDstV6 := fmt.Sprintf("svc_port_%s_dst_v6", slugName)
	chainForward := fmt.Sprintf("svc_port_%s_forward", slugName)

	// Build nft batch script for atomic execution
	var batch strings.Builder
	batch.WriteString("table inet internet {\n")

	// Create sets for client MACs and IPs (for return traffic)
	batch.WriteString(fmt.Sprintf("\tset %s {\n\t\ttype ether_addr\n\t}\n", setMacs))
	batch.WriteString(fmt.Sprintf("\tset %s {\n\t\ttype ipv4_addr\n\t}\n", setClientIpsV4))
	batch.WriteString(fmt.Sprintf("\tset %s {\n\t\ttype ipv6_addr\n\t}\n", setClientIpsV6))

	// Create sets for destination IPs if provided
	hasDstIPs := len(dstIPs.IPv4) > 0 || len(dstIPs.IPv6) > 0
	if hasDstIPs {
		batch.WriteString(fmt.Sprintf("\tset %s {\n\t\ttype ipv4_addr\n\t}\n", setDstV4))
		batch.WriteString(fmt.Sprintf("\tset %s {\n\t\ttype ipv6_addr\n\t}\n", setDstV6))
	}

	// Create forward chain with outbound + return rules.
	// Uses explicit client IP sets for return traffic (no conntrack).
	// Same pattern as DstIpGroup for consistency and zero ct overhead.
	batch.WriteString(fmt.Sprintf("\tchain %s {\n", chainForward))
	for _, proto := range protocols {
		proto = strings.ToLower(proto)
		if hasDstIPs {
			// Outbound: client -> destination on service port (with destination IP restrictions)
			batch.WriteString(fmt.Sprintf("\t\tether saddr @%s ip daddr @%s %s dport %d counter accept\n",
				setMacs, setDstV4, proto, port))
			batch.WriteString(fmt.Sprintf("\t\tether saddr @%s ip6 daddr @%s %s dport %d counter accept\n",
				setMacs, setDstV6, proto, port))
			// Return: destination -> client from service port (with source IP restrictions)
			batch.WriteString(fmt.Sprintf("\t\tip saddr @%s ip daddr @%s %s sport %d counter accept\n",
				setDstV4, setClientIpsV4, proto, port))
			batch.WriteString(fmt.Sprintf("\t\tip6 saddr @%s ip6 daddr @%s %s sport %d counter accept\n",
				setDstV6, setClientIpsV6, proto, port))
		} else {
			// Outbound: client -> any destination on service port (no destination IP restrictions)
			batch.WriteString(fmt.Sprintf("\t\tether saddr @%s %s dport %d counter accept\n",
				setMacs, proto, port))
			// Return: any source -> client from service port (no source IP restrictions)
			batch.WriteString(fmt.Sprintf("\t\tip daddr @%s %s sport %d counter accept\n",
				setClientIpsV4, proto, port))
			batch.WriteString(fmt.Sprintf("\t\tip6 daddr @%s %s sport %d counter accept\n",
				setClientIpsV6, proto, port))
		}
	}
	batch.WriteString("\t}\n")

	batch.WriteString("}\n")

	// Add destination IPs to sets if provided
	if len(dstIPs.IPv4) > 0 {
		ipList := strings.Join(dstIPs.IPv4, ", ")
		batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setDstV4, ipList))
	}
	if len(dstIPs.IPv6) > 0 {
		ipList := strings.Join(dstIPs.IPv6, ", ")
		batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setDstV6, ipList))
	}

	// Execute batch command using nft -f - with heredoc for safe shell escaping
	cmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
	if err := shell.Exec(cmd, nil); err != nil {
		return fmt.Errorf("failed to create service port: %v", err)
	}

	// Append jump rule to end of forward chain (after authenticated users are handled)
	// Service port jumps allow unauthenticated clients limited access to specific services
	jumpCmd := fmt.Sprintf("nft add rule inet internet forward counter jump %s", chainForward)
	if err := shell.Exec(jumpCmd, nil); err != nil {
		// Cleanup: delete the chain we just created
		shell.Exec(fmt.Sprintf("nft delete chain inet internet %s 2>/dev/null || true", chainForward), nil)
		return fmt.Errorf("failed to append jump rule: %v", err)
	}

	// Store service port definition (only after nftables success)
	lowerProtos := make([]string, len(protocols))
	for i, proto := range protocols {
		lowerProtos[i] = strings.ToLower(proto)
	}

	self.firewallMutex.Lock()
	self.createdServicePorts[slugName] = true
	self.servicePortDefs[slugName] = servicePortDefinition{
		Protocols: lowerProtos,
		Port:      port,
		DstIPv4:   dstIPs.IPv4,
		DstIPv6:   dstIPs.IPv6,
	}
	self.firewallMutex.Unlock()

	return nil
}

// ServicePortExists checks if a named service port exists.
func (self *FirewallApi) ServicePortExists(name string) (bool, error) {
	// Slugify the service port name to match how it was created
	slugName := sdkutils.Slugify(name, "_")
	if slugName == "" {
		return false, fmt.Errorf("invalid service port name: %s (must contain alphanumeric characters)", name)
	}

	self.firewallMutex.RLock()
	exists := self.createdServicePorts[slugName]
	self.firewallMutex.RUnlock()

	return exists, nil
}

// DeleteServicePort removes a named service port and all its nftables infrastructure.
// All clients currently allowed access through this service port will immediately lose access.
func (self *FirewallApi) DeleteServicePort(name string) error {
	// Slugify the service port name to match how it was created
	slugName := sdkutils.Slugify(name, "_")
	if slugName == "" {
		return fmt.Errorf("invalid service port name: %s (must contain alphanumeric characters)", name)
	}

	contextInfo := fmt.Sprintf("ServicePortName=%s (slug=%s)", name, slugName)

	_, err := self.firewallQue.ExecWithTimeout(
		10*time.Second,
		"Delete Service Port",
		contextInfo,
		func() (any, error) {
			return nil, self.doDeleteServicePort(slugName)
		},
	)
	return err
}

// doDeleteServicePort is the internal implementation of DeleteServicePort
func (self *FirewallApi) doDeleteServicePort(slugName string) error {
	// Check if service port exists
	self.firewallMutex.RLock()
	if !self.createdServicePorts[slugName] {
		self.firewallMutex.RUnlock()
		return fmt.Errorf("service port does not exist: %s", slugName)
	}
	self.firewallMutex.RUnlock()

	// Cancel all timers for this service port
	self.cancelTimersWithPrefix("svc:" + slugName + ":")

	// Define nftables resource names
	chainForward := fmt.Sprintf("svc_port_%s_forward", slugName)
	setMacs := fmt.Sprintf("svc_port_%s_macs", slugName)
	setClientIpsV4 := fmt.Sprintf("svc_port_%s_client_ips_v4", slugName)
	setClientIpsV6 := fmt.Sprintf("svc_port_%s_client_ips_v6", slugName)
	setDstV4 := fmt.Sprintf("svc_port_%s_dst_v4", slugName)
	setDstV6 := fmt.Sprintf("svc_port_%s_dst_v6", slugName)

	// Delete jump rule from forward chain
	self.deleteJumpRule("forward", chainForward)

	// Build batch script to delete chain and sets
	var batch strings.Builder

	// Flush and delete forward chain
	batch.WriteString(fmt.Sprintf("flush chain inet internet %s 2>/dev/null || true\n", chainForward))
	batch.WriteString(fmt.Sprintf("delete chain inet internet %s 2>/dev/null || true\n", chainForward))

	// Delete sets
	batch.WriteString(fmt.Sprintf("delete set inet internet %s 2>/dev/null || true\n", setMacs))
	batch.WriteString(fmt.Sprintf("delete set inet internet %s 2>/dev/null || true\n", setClientIpsV4))
	batch.WriteString(fmt.Sprintf("delete set inet internet %s 2>/dev/null || true\n", setClientIpsV6))
	batch.WriteString(fmt.Sprintf("delete set inet internet %s 2>/dev/null || true\n", setDstV4))
	batch.WriteString(fmt.Sprintf("delete set inet internet %s 2>/dev/null || true\n", setDstV6))

	// Execute batch command
	cmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
	if err := shell.Exec(cmd, nil); err != nil {
		return fmt.Errorf("failed to delete service port infrastructure: %v", err)
	}

	// Remove from tracking maps
	self.firewallMutex.Lock()
	delete(self.createdServicePorts, slugName)
	delete(self.servicePortDefs, slugName)
	self.firewallMutex.Unlock()

	return nil
}

// AllowClientToServicePort allows a specific client device to access a named service port.
func (self *FirewallApi) AllowClientToServicePort(clnt sdkapi.DstIpGroupClient, servicePortName string, timeoutSecs int) error {
	// Slugify the service port name to match how it was created
	slugName := sdkutils.Slugify(servicePortName, "_")
	if slugName == "" {
		return fmt.Errorf("invalid service port name: %s (must contain alphanumeric characters)", servicePortName)
	}

	contextInfo := fmt.Sprintf("ServicePortName=%s, ClientMAC=%s, ClientIP=%s", servicePortName, clnt.MacAddr, clnt.IpAddr)

	_, err := self.firewallQue.ExecWithTimeout(
		5*time.Second,
		"Allow Client to Service Port",
		contextInfo,
		func() (any, error) {
			return nil, self.doAllowClientToServicePort(clnt, slugName, timeoutSecs)
		},
	)
	return err
}

// doAllowClientToServicePort is the internal implementation of AllowClientToServicePort
func (self *FirewallApi) doAllowClientToServicePort(clnt sdkapi.DstIpGroupClient, slugName string, timeoutSecs int) error {
	// Check if service port exists
	self.firewallMutex.RLock()
	if !self.createdServicePorts[slugName] {
		self.firewallMutex.RUnlock()
		return fmt.Errorf("service port does not exist: %s", slugName)
	}
	self.firewallMutex.RUnlock()

	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(clnt.MacAddr)
	if err != nil {
		return fmt.Errorf("MAC validation failed: %v", err)
	}
	clnt.MacAddr = normalizedMAC

	// Resolve effective IPv4/IPv6 addresses (same logic as DstIpGroup)
	ipv4, ipv6 := clnt.Ipv4Addr, clnt.Ipv6Addr
	if ipv4 == "" && ipv6 == "" && clnt.IpAddr != "" {
		ver, verErr := sdkutils.GetIPVersion(clnt.IpAddr)
		if verErr == nil {
			if ver == "ip" {
				ipv4 = clnt.IpAddr
			} else {
				ipv6 = clnt.IpAddr
			}
		}
	}

	// Validate each IP address that is present
	if ipv4 != "" {
		if _, err := sdkutils.ValidateIPAddress(ipv4); err != nil {
			return fmt.Errorf("client IPv4 validation failed: %v", err)
		}
	}
	if ipv6 != "" {
		if _, err := sdkutils.ValidateIPAddress(ipv6); err != nil {
			return fmt.Errorf("client IPv6 validation failed: %v", err)
		}
	}

	// Define nftables set names
	setMacs := fmt.Sprintf("svc_port_%s_macs", slugName)
	setClientIpsV4 := fmt.Sprintf("svc_port_%s_client_ips_v4", slugName)
	setClientIpsV6 := fmt.Sprintf("svc_port_%s_client_ips_v6", slugName)

	// Cancel any existing timer for this client in this service port
	cacheKey := fmt.Sprintf("svc:%s:%s", slugName, clnt.MacAddr)
	self.firewallMutex.Lock()
	if existingTimer, ok := self.activeTimers[cacheKey]; ok {
		existingTimer.Stop()
		delete(self.activeTimers, cacheKey)
	}
	self.firewallMutex.Unlock()

	// Build nft batch script to add client MAC and both IP addresses (where present)
	var batch strings.Builder
	batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setMacs, clnt.MacAddr))
	if ipv4 != "" {
		batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setClientIpsV4, ipv4))
	}
	if ipv6 != "" {
		batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setClientIpsV6, ipv6))
	}

	// Execute batch command using heredoc for safe shell escaping
	nftCmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
	if err := shell.Exec(nftCmd, nil); err != nil {
		return fmt.Errorf("failed to add client to service port: %v", err)
	}

	// Schedule automatic removal if timeout is specified
	if timeoutSecs > 0 {
		self.scheduleServicePortClientRemoval(slugName, clnt, timeoutSecs)
	}

	return nil
}

// RemoveClientFromServicePort removes access for a specific client device from a named service port.
func (self *FirewallApi) RemoveClientFromServicePort(clnt sdkapi.DstIpGroupClient, servicePortName string) error {
	// Slugify the service port name to match how it was created
	slugName := sdkutils.Slugify(servicePortName, "_")
	if slugName == "" {
		return fmt.Errorf("invalid service port name: %s (must contain alphanumeric characters)", servicePortName)
	}

	contextInfo := fmt.Sprintf("ServicePortName=%s, ClientMAC=%s", servicePortName, clnt.MacAddr)

	_, err := self.firewallQue.ExecWithTimeout(
		5*time.Second,
		"Remove Client from Service Port",
		contextInfo,
		func() (any, error) {
			return nil, self.doRemoveClientFromServicePort(clnt, slugName)
		},
	)
	return err
}

// doRemoveClientFromServicePort is the internal implementation of RemoveClientFromServicePort
func (self *FirewallApi) doRemoveClientFromServicePort(clnt sdkapi.DstIpGroupClient, slugName string) error {
	// Check if service port exists
	self.firewallMutex.RLock()
	if !self.createdServicePorts[slugName] {
		self.firewallMutex.RUnlock()
		return fmt.Errorf("service port does not exist: %s", slugName)
	}
	self.firewallMutex.RUnlock()

	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(clnt.MacAddr)
	if err != nil {
		return fmt.Errorf("MAC validation failed: %v", err)
	}
	clnt.MacAddr = normalizedMAC

	// Resolve effective IPv4/IPv6 addresses (same logic as DstIpGroup)
	ipv4, ipv6 := clnt.Ipv4Addr, clnt.Ipv6Addr
	if ipv4 == "" && ipv6 == "" && clnt.IpAddr != "" {
		ver, verErr := sdkutils.GetIPVersion(clnt.IpAddr)
		if verErr == nil {
			if ver == "ip" {
				ipv4 = clnt.IpAddr
			} else {
				ipv6 = clnt.IpAddr
			}
		}
	}

	// Cancel any active timer for this client in this service port
	cacheKey := fmt.Sprintf("svc:%s:%s", slugName, clnt.MacAddr)
	self.firewallMutex.Lock()
	if timer, ok := self.activeTimers[cacheKey]; ok {
		timer.Stop()
		delete(self.activeTimers, cacheKey)
	}
	self.firewallMutex.Unlock()

	// Define nftables set names
	setMacs := fmt.Sprintf("svc_port_%s_macs", slugName)
	setClientIpsV4 := fmt.Sprintf("svc_port_%s_client_ips_v4", slugName)
	setClientIpsV6 := fmt.Sprintf("svc_port_%s_client_ips_v6", slugName)

	// Remove MAC and both IP addresses (best-effort — elements may already be absent).
	shell.Exec(fmt.Sprintf("nft delete element inet internet %s '{ %s }' 2>/dev/null || true", setMacs, clnt.MacAddr), nil)
	if ipv4 != "" {
		shell.Exec(fmt.Sprintf("nft delete element inet internet %s '{ %s }' 2>/dev/null || true", setClientIpsV4, ipv4), nil)
	}
	if ipv6 != "" {
		shell.Exec(fmt.Sprintf("nft delete element inet internet %s '{ %s }' 2>/dev/null || true", setClientIpsV6, ipv6), nil)
	}

	return nil
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// validateServicePortParams validates protocols and port parameters
func (self *FirewallApi) validateServicePortParams(protocols []string, port int) error {
	if len(protocols) == 0 {
		return fmt.Errorf("at least one protocol is required")
	}
	for _, proto := range protocols {
		proto = strings.ToLower(proto)
		if proto != "tcp" && proto != "udp" {
			return fmt.Errorf("invalid protocol %q: must be 'tcp' or 'udp'", proto)
		}
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
	}
	return nil
}

// scheduleServicePortClientRemoval schedules automatic removal of a client's service port access
func (self *FirewallApi) scheduleServicePortClientRemoval(slugName string, clnt sdkapi.DstIpGroupClient, timeoutSecs int) {
	cacheKey := fmt.Sprintf("svc:%s:%s", slugName, clnt.MacAddr)

	timer := time.AfterFunc(time.Duration(timeoutSecs)*time.Second, func() {
		// Remove timer from tracking map
		self.firewallMutex.Lock()
		delete(self.activeTimers, cacheKey)
		self.firewallMutex.Unlock()

		// Remove the client's service port access
		// Note: We need the original service port name, but we only have slugName
		// For simplicity, we'll use slugName directly (it matches what the method expects after slugification)
		err := self.RemoveClientFromServicePort(clnt, slugName)
		if err != nil {
			// Log error but don't panic - this is a background operation
			fmt.Printf("Warning: Failed to auto-remove client %s from service port %s: %v\n", clnt.MacAddr, slugName, err)
		}
	})

	// Store timer in tracking map
	self.firewallMutex.Lock()
	self.activeTimers[cacheKey] = timer
	self.firewallMutex.Unlock()
}

// cancelTimersWithPrefix cancels all timers whose keys start with the given prefix.
// Used when deleting groups or service ports to cancel all pending client removals.
func (self *FirewallApi) cancelTimersWithPrefix(prefix string) {
	self.firewallMutex.Lock()
	defer self.firewallMutex.Unlock()

	keysToDelete := []string{}
	for key, timer := range self.activeTimers {
		if strings.HasPrefix(key, prefix) {
			timer.Stop()
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(self.activeTimers, key)
	}
}

// deleteJumpRule finds and deletes a jump rule from mainChain to targetChain.
// Uses JSON parsing to find the rule handle, then deletes by handle.
func (self *FirewallApi) deleteJumpRule(mainChain, targetChain string) {
	// Use nft -j -a to list rules with handles as JSON
	var out bytes.Buffer
	cmd := fmt.Sprintf("nft -j -a list chain inet internet %s", mainChain)
	if err := shell.ExecOutput(cmd, &out); err != nil {
		return // Best effort - chain may not exist
	}

	// Parse JSON and find matching jump rule handles
	handles := self.findJumpRuleHandles(out.Bytes(), targetChain)

	// Delete each matching rule by handle
	for _, handle := range handles {
		shell.Exec(fmt.Sprintf("nft delete rule inet internet %s handle %d 2>/dev/null || true", mainChain, handle), nil)
	}
}

// findJumpRuleHandles parses nft JSON output and returns handles of jump rules to targetChain.
// A rule matches if it contains a jump verdict to the target chain.
func (self *FirewallApi) findJumpRuleHandles(jsonData []byte, targetChain string) []int {
	var result nftRuleListResult
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil
	}

	var handles []int

	for _, entry := range result.Nftables {
		if entry.Rule == nil {
			continue
		}
		rule := entry.Rule

		// Check if this rule contains a jump to targetChain
		if self.ruleContainsJump(rule.Expr, targetChain) {
			handles = append(handles, rule.Handle)
		}
	}

	return handles
}

// ruleContainsJump checks if a rule's expressions contain a jump verdict to the target chain.
func (self *FirewallApi) ruleContainsJump(exprs []interface{}, targetChain string) bool {
	for _, expr := range exprs {
		exprMap, ok := expr.(map[string]interface{})
		if !ok {
			continue
		}

		// Check for "jump" expression
		jumpExpr, ok := exprMap["jump"].(map[string]interface{})
		if !ok {
			continue
		}

		// Check the target
		target, ok := jumpExpr["target"].(string)
		if ok && target == targetChain {
			return true
		}
	}

	return false
}

var _ sdkapi.IFirewallAPI = (*FirewallApi)(nil)
