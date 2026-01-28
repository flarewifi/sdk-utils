package api

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	jobque "core/utils/job-que"
	"core/utils/shell"
	sdkutils "github.com/flarehotspot/sdk-utils"
	sdkapi "sdk/api"
)

const (
	// DstIpGroupMaxAge is the maximum age for IPs in a destination group before they're flushed
	DstIpGroupMaxAge = 12 * time.Hour
)

func NewFirewallApi(api *PluginApi) {
	firewallApi := &FirewallApi{
		activeTimers:  make(map[string]*time.Timer),
		firewallMutex: &sync.RWMutex{},
		firewallQue:   jobque.NewJobQue[any](),
		createdGroups: make(map[string]bool),
		groupIPs:      make(map[string]map[string]time.Time),
	}
	api.FirewallAPI = firewallApi
}

type FirewallApi struct {
	activeTimers  map[string]*time.Timer          // Track active removal timers by "destIP:mac" key
	firewallMutex *sync.RWMutex                   // Protect concurrent access to activeTimers, createdGroups, and groupIPs
	firewallQue   *jobque.JobQue[any]             // Serialize firewall operations to prevent race conditions
	createdGroups map[string]bool                 // Track created destination IP groups by slugified name
	groupIPs      map[string]map[string]time.Time // Track IPs per group with timestamp when added (slugName -> IP -> addedAt)
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

	// Add jump rules from main chains to group chains
	batch.WriteString(fmt.Sprintf("insert rule inet internet prerouting counter jump %s\n", chainPrerouting))
	batch.WriteString(fmt.Sprintf("insert rule inet internet forward counter jump %s\n", chainForward))

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

	// Validate client IP address
	if _, err := sdkutils.ValidateIPAddress(clnt.IpAddr); err != nil {
		return fmt.Errorf("client IP validation failed: %v", err)
	}

	// Determine IP version for client
	clientIpVersion, err := sdkutils.GetIPVersion(clnt.IpAddr)
	if err != nil {
		return fmt.Errorf("failed to determine client IP version: %v", err)
	}

	// Define nftables set names
	setMacs := fmt.Sprintf("dst_grp_%s_macs", slugName)
	var setClientIps string
	if clientIpVersion == "ip" {
		setClientIps = fmt.Sprintf("dst_grp_%s_client_ips_v4", slugName)
	} else {
		setClientIps = fmt.Sprintf("dst_grp_%s_client_ips_v6", slugName)
	}

	// Create cache key for tracking timers
	cacheKey := fmt.Sprintf("grp:%s:%s", slugName, clnt.MacAddr)

	// Cancel any existing timer for this client in this group
	self.firewallMutex.Lock()
	if existingTimer, ok := self.activeTimers[cacheKey]; ok {
		existingTimer.Stop()
		delete(self.activeTimers, cacheKey)
	}
	self.firewallMutex.Unlock()

	// Build nft batch script to add client to sets
	var batch strings.Builder
	batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setMacs, clnt.MacAddr))
	batch.WriteString(fmt.Sprintf("add element inet internet %s { %s }\n", setClientIps, clnt.IpAddr))

	// Execute batch command using heredoc for safe shell escaping
	cmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
	if err := shell.Exec(cmd, nil); err != nil {
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

	// Validate client IP address
	if _, err := sdkutils.ValidateIPAddress(clnt.IpAddr); err != nil {
		return fmt.Errorf("client IP validation failed: %v", err)
	}

	// Determine IP version for client
	clientIpVersion, err := sdkutils.GetIPVersion(clnt.IpAddr)
	if err != nil {
		return fmt.Errorf("failed to determine client IP version: %v", err)
	}

	// Define nftables set names
	setMacs := fmt.Sprintf("dst_grp_%s_macs", slugName)
	var setClientIps string
	if clientIpVersion == "ip" {
		setClientIps = fmt.Sprintf("dst_grp_%s_client_ips_v4", slugName)
	} else {
		setClientIps = fmt.Sprintf("dst_grp_%s_client_ips_v6", slugName)
	}

	// Cancel any active timer for this client in this group
	cacheKey := fmt.Sprintf("grp:%s:%s", slugName, clnt.MacAddr)
	self.firewallMutex.Lock()
	if timer, ok := self.activeTimers[cacheKey]; ok {
		timer.Stop()
		delete(self.activeTimers, cacheKey)
	}
	self.firewallMutex.Unlock()

	// Build nft batch script to remove client from sets
	var batch strings.Builder
	batch.WriteString(fmt.Sprintf("delete element inet internet %s { %s }\n", setMacs, clnt.MacAddr))
	batch.WriteString(fmt.Sprintf("delete element inet internet %s { %s }\n", setClientIps, clnt.IpAddr))

	// Execute batch command using heredoc for safe shell escaping
	cmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
	if err := shell.Exec(cmd, nil); err != nil {
		// Log warning but don't fail - element may have already been removed
		log.Printf("Warning: Failed to remove client %s from group %s (may already be removed): %v", clnt.MacAddr, slugName, err)
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

var _ sdkapi.IFirewallAPI = (*FirewallApi)(nil)
