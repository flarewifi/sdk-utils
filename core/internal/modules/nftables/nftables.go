//go:build !dev

package nftables

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"core/utils/arp"
	jobque "core/utils/job-que"
	cmd "core/utils/shell"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// nftRuleList parses the relevant subset of "nft -j -a list chain" output so we
// can find rule handles to delete (used by deleteCaptiveDnatRules).
type nftRuleList struct {
	Nftables []struct {
		Rule *struct {
			Handle int           `json:"handle"`
			Expr   []interface{} `json:"expr"`
		} `json:"rule,omitempty"`
	} `json:"nftables"`
}

const (
	internetTable    string = "internet" // Our custom table
	tableFamily      string = "inet"     // inet family (handles both ipv4 and ipv6)
	forwardChain     string = "forward"
	preroutingChain  string = "prerouting"
	postroutingChain string = "postrouting"
	connMacMap       string = "connected_macs_map"
	connIpMap        string = "connected_ips_map"
	connIp6Map       string = "connected_ips6_map"
	connMacSet          string = "connected_macs_set"
	whitelistMacSet     string = "whitelist_macs"
	whitelistIpsV4Set   string = "whitelist_client_ips_v4"
	whitelistIpsV6Set   string = "whitelist_client_ips_v6"
	whitelistForward    string = "whitelist_forward"
	whitelistPrerouting string = "whitelist_prerouting"

	// Hard-block sets: an absolute deny that sits ABOVE the session and whitelist
	// accepts in the forward chain, so a blocked device loses internet regardless
	// of whether it has an active session or is whitelisted.
	blockedMacSet   string = "blocked_macs"
	blockedIpsV4Set string = "blocked_client_ips_v4"
	blockedIpsV6Set string = "blocked_client_ips_v6"

	// managedIfacesSet (type ifname) holds the interfaces the app manages.
	// Membership is toggled by SetInterfaceMode as the admin flips an interface
	// between managed and unmanaged on the Interfaces page — no rule surgery
	// needed, so changes apply instantly. Session enforcement and anti-tether are
	// scoped to this set; traffic that touches NO managed interface is passed
	// straight through (see Setup), so unmanaged interfaces are left untouched.
	managedIfacesSet string = "managed_ifaces"

	// captiveIfacesSet (type ifname) is the subset of managed interfaces that also
	// run the captive-portal redirect. It drives the port-80 DNAT rule ONLY, so an
	// interface can be managed (session-gated) without auto-redirecting clients to
	// the portal. Always a subset of managedIfacesSet (see IsCaptive).
	captiveIfacesSet string = "captive_ifaces"

	// portalIpsV4Set/portalIpsV6Set hold the portal-serving addresses from
	// interfaces.json: the gateway IP of every captive-portal-enabled LAN plus the
	// main portal interface's IP (see SetPortalIPs). Traffic from a captive
	// interface destined to any of these bypasses BOTH the prerouting port-80 DNAT
	// (so http://<captive-gateway-ip> is served as addressed, never rewritten to
	// the main IP) and the forward chain's session gate (so an unauthenticated
	// client on one captive subnet can always reach the portal hosted on
	// another's address).
	portalIpsV4Set string = "portal_ips_v4"
	portalIpsV6Set string = "portal_ips_v6"

	// whitelistReconcileInterval is how often the background reconciler re-syncs
	// each whitelisted MAC's current IP (from the ARP/neighbor table) into the
	// return-traffic sets. This is the session-independent safety net that
	// converges whitelisted devices which changed IP without ever firing a
	// Connect event (e.g. a no-session device whose DHCP lease rebound).
	whitelistReconcileInterval = 30 * time.Second
)

var (
	nftMu  sync.RWMutex
	nftQue = jobque.NewJobQueue[any]()

	// ipToMac maps connected IP address (IPv4 or IPv6) → uppercase normalized MAC.
	// Populated on Connect, evicted on Disconnect.
	// Guarded by nftMu.
	ipToMac = make(map[string]string)

	// macToIps maps connected MAC → set of IPs registered for that device.
	// Populated on Connect, evicted on Disconnect.
	// Guarded by nftMu.
	macToIps = make(map[string]map[string]bool)

	// whitelistedMacs tracks MACs allowed via AllowMAC (the upload-side bypass).
	// Used so Connect/Disconnect can keep whitelist_client_ips_* (the
	// download/return side) in sync with each whitelisted device's live IP as it
	// connects, disconnects, or changes IP. Guarded by nftMu.
	whitelistedMacs = make(map[string]bool)

	// whitelistMacIps maps a whitelisted MAC → set of client IPs currently
	// registered in whitelist_client_ips_* for that MAC. Lets DisallowMAC remove
	// exactly what was added, and the reconciler prune stale IPs. Guarded by nftMu.
	whitelistMacIps = make(map[string]map[string]bool)

	// blockedMacs tracks MACs hard-blocked via BlockMAC (the absolute upload-side
	// deny). The MAC drop is IP-independent and permanent until UnblockMAC, so this
	// is the source of truth for the block. Guarded by nftMu.
	blockedMacs = make(map[string]bool)

	// blockedMacIps maps a blocked MAC → set of client IPs added to
	// blocked_client_ips_* (the download/return-side deny captured at block time).
	// Lets UnblockMAC remove exactly what was added. Guarded by nftMu.
	blockedMacIps = make(map[string]map[string]bool)

	// whitelistReconcilerOnce ensures the background reconciler goroutine is
	// started at most once, even if Setup runs again.
	whitelistReconcilerOnce sync.Once
)

func Cleanup() {
	cmds := []string{
		// Delete our custom table (this removes all chains, maps, sets, and rules within it)
		fmt.Sprintf("nft delete table %s %s 2>/dev/null || true", tableFamily, internetTable),
	}
	cmd.ExecAll(cmds)
}

func Setup() (err error) {
	Cleanup()

	// Build nft batch script for atomic execution
	var batch strings.Builder

	// Create our custom internet table
	batch.WriteString(fmt.Sprintf("add table %s %s\n", tableFamily, internetTable))

	// Create custom forward and prerouting chains as base chains with hooks (priority -1 runs before fw4)
	batch.WriteString(fmt.Sprintf("add chain %s %s %s { type filter hook forward priority -250; policy drop; }\n", tableFamily, internetTable, forwardChain))
	batch.WriteString(fmt.Sprintf("add chain %s %s %s { type nat hook prerouting priority -1; policy accept; }\n", tableFamily, internetTable, preroutingChain))

	// Create maps and sets in our custom table
	// IPv4 verdict map (download traffic: accept + accounting)
	batch.WriteString(fmt.Sprintf("add map %s %s %s { type ipv4_addr : verdict ; counter; }\n", tableFamily, internetTable, connIpMap))
	// IPv6 verdict map (download traffic: accept + accounting)
	batch.WriteString(fmt.Sprintf("add map %s %s %s { type ipv6_addr : verdict ; counter; }\n", tableFamily, internetTable, connIp6Map))
	// MAC verdict map and set (protocol-agnostic)
	batch.WriteString(fmt.Sprintf("add map %s %s %s { type ether_addr : verdict ; counter; }\n", tableFamily, internetTable, connMacMap))
	batch.WriteString(fmt.Sprintf("add set %s %s %s { type ether_addr; }\n", tableFamily, internetTable, connMacSet))

	// Hard-block sets (absolute deny — see BlockMAC). Declared here so the drop
	// rules below, which sit at the very top of the forward chain, can reference them.
	batch.WriteString(fmt.Sprintf("add set %s %s %s { type ether_addr; }\n", tableFamily, internetTable, blockedMacSet))
	batch.WriteString(fmt.Sprintf("add set %s %s %s { type ipv4_addr; }\n", tableFamily, internetTable, blockedIpsV4Set))
	batch.WriteString(fmt.Sprintf("add set %s %s %s { type ipv6_addr; }\n", tableFamily, internetTable, blockedIpsV6Set))

	// Managed-interface set (see SetInterfaceMode). Declared here so the
	// transparency-passthrough forward rule and the anti-tether postrouting rules
	// below can reference it; membership starts empty and is filled by
	// ApplyPortalConfig.
	batch.WriteString(fmt.Sprintf("add set %s %s %s { type ifname; }\n", tableFamily, internetTable, managedIfacesSet))
	// Captive-interface set: the subset of managed interfaces whose port-80
	// traffic is DNAT'd to the portal (see SetCaptivePortalTarget).
	batch.WriteString(fmt.Sprintf("add set %s %s %s { type ifname; }\n", tableFamily, internetTable, captiveIfacesSet))
	// Portal-IP sets: the portal-serving addresses from interfaces.json (see
	// SetPortalIPs). Referenced by the portal bypass rules in the forward and
	// prerouting chains below; membership is filled by ApplyPortalConfig.
	batch.WriteString(fmt.Sprintf("add set %s %s %s { type ipv4_addr; }\n", tableFamily, internetTable, portalIpsV4Set))
	batch.WriteString(fmt.Sprintf("add set %s %s %s { type ipv6_addr; }\n", tableFamily, internetTable, portalIpsV6Set))

	// Create postrouting chain for anti-tethering (TTL set)
	// Sets outgoing TTL to 1 so tethered devices cannot forward packets (TTL drops to 0)
	batch.WriteString(fmt.Sprintf("add chain %s %s %s { type filter hook postrouting priority 0; policy accept; }\n", tableFamily, internetTable, postroutingChain))

	// Anti-tethering applies ONLY to managed interfaces: set TTL/hoplimit=1 on
	// packets egressing through any interface in managed_ifaces. Unmanaged
	// interfaces are intentionally excluded so they route normally.
	batch.WriteString(fmt.Sprintf("add rule %s %s %s oifname @%s ip ttl set 1\n", tableFamily, internetTable, postroutingChain, managedIfacesSet))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s oifname @%s ip6 hoplimit set 1\n", tableFamily, internetTable, postroutingChain, managedIfacesSet))

	// Add rules to our custom forward chain.
	//
	// Verdict-map-only design: no conntrack lookups, O(1) hash table matches.
	// This avoids per-packet conntrack overhead that causes latency for gaming.
	//
	// Rule order (first terminal verdict wins):
	//   0. Hard block: source MAC / dest client IP in blocked sets — DROP. These
	//      come FIRST so an absolute block beats the session and whitelist accepts.
	//   0a. Portal bypass: ingress captive AND destination in the portal-IP sets —
	//      ACCEPT. A client on any captive subnet must always be able to reach the
	//      portal-serving interface IPs (interfaces.json), session or not — the
	//      portal IS how it gets a session. Placed after the hard-block drops so
	//      a blocked device still can't reach anything.
	//   0b. Inter-captive routing: ingress AND egress both managed (captive) —
	//      ACCEPT. This is LAN-to-LAN traffic between the app's own subnets, so
	//      captive interfaces can reach each other without a session. Internet
	//      egress leaves via the unmanaged WAN, so it does NOT match and still
	//      falls through to the verdict maps below (stays session-gated).
	//   1. Upload: source MAC verdict map — accept if MAC is registered.
	//   2. Download (IPv4): destination IP verdict map — accept + count.
	//   3. Download (IPv6): destination IP6 verdict map — accept + count.
	//   4. (implicit) chain policy drop — everything else is blocked.
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ether saddr @%s counter drop\n", tableFamily, internetTable, forwardChain, blockedMacSet))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ip daddr @%s counter drop\n", tableFamily, internetTable, forwardChain, blockedIpsV4Set))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ip6 daddr @%s counter drop\n", tableFamily, internetTable, forwardChain, blockedIpsV6Set))
	// Portal bypass: forwarded traffic from a captive interface to any
	// portal-serving IP is accepted unconditionally (no session needed). In the
	// common case this traffic is locally delivered (INPUT) and never forwarded,
	// but any topology that DOES forward it must not have the portal blackholed
	// by the policy drop below.
	batch.WriteString(fmt.Sprintf("add rule %s %s %s iifname @%s ip daddr @%s counter accept\n", tableFamily, internetTable, forwardChain, captiveIfacesSet, portalIpsV4Set))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s iifname @%s ip6 daddr @%s counter accept\n", tableFamily, internetTable, forwardChain, captiveIfacesSet, portalIpsV6Set))
	// Transparency passthrough: accept any forwarded packet that touches NO
	// managed interface (neither ingress nor egress), so unmanaged interfaces flow
	// straight through to the system's own firewall — our policy-drop chain stays
	// invisible to them. Placed AFTER the hard-block drops (so an explicit block
	// still wins everywhere) but BEFORE the session verdict maps (so unmanaged
	// traffic never needs a session).
	batch.WriteString(fmt.Sprintf("add rule %s %s %s iifname != @%s oifname != @%s counter accept\n", tableFamily, internetTable, forwardChain, managedIfacesSet, managedIfacesSet))
	// Inter-captive routing: both ingress and egress are managed (captive)
	// interfaces — accept so the app's own LAN subnets can reach each other
	// without a session. Internet egress (managed → unmanaged WAN) does NOT match
	// this and falls through to the session verdict maps below, staying gated.
	batch.WriteString(fmt.Sprintf("add rule %s %s %s iifname @%s oifname @%s counter accept\n", tableFamily, internetTable, forwardChain, managedIfacesSet, managedIfacesSet))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ether saddr vmap @%s\n", tableFamily, internetTable, forwardChain, connMacMap))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ip daddr vmap @%s\n", tableFamily, internetTable, forwardChain, connIpMap))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ip6 daddr vmap @%s\n", tableFamily, internetTable, forwardChain, connIp6Map))
	// Service port jumps will be appended after this rule for unauthenticated clients.

	// Whitelist sets and chains (used by the whitelist plugin for MAC-based bypass)
	batch.WriteString(fmt.Sprintf("add set %s %s %s { type ether_addr; }\n", tableFamily, internetTable, whitelistMacSet))
	batch.WriteString(fmt.Sprintf("add set %s %s %s { type ipv4_addr; flags timeout; }\n", tableFamily, internetTable, whitelistIpsV4Set))
	batch.WriteString(fmt.Sprintf("add set %s %s %s { type ipv6_addr; flags timeout; }\n", tableFamily, internetTable, whitelistIpsV6Set))
	batch.WriteString(fmt.Sprintf("add chain %s %s %s\n", tableFamily, internetTable, whitelistForward))
	batch.WriteString(fmt.Sprintf("add chain %s %s %s\n", tableFamily, internetTable, whitelistPrerouting))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ether saddr @%s counter accept\n", tableFamily, internetTable, whitelistForward, whitelistMacSet))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ip daddr @%s counter accept\n", tableFamily, internetTable, whitelistForward, whitelistIpsV4Set))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ip6 daddr @%s counter accept\n", tableFamily, internetTable, whitelistForward, whitelistIpsV6Set))
	batch.WriteString(fmt.Sprintf("add rule %s %s %s ether saddr @%s counter accept\n", tableFamily, internetTable, whitelistPrerouting, whitelistMacSet))

	// Execute batch command using nft -f - with heredoc for atomic execution
	nftCmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
	err = cmd.Exec(nftCmd, nil)
	if err != nil {
		return err
	}

	// Jump rules added after batch — append whitelist_forward (after verdict maps),
	// insert whitelist_prerouting (before captive portal DNAT).
	err = cmd.Exec(fmt.Sprintf("nft add rule %s %s %s counter jump %s", tableFamily, internetTable, forwardChain, whitelistForward), nil)
	if err != nil {
		return err
	}
	err = cmd.Exec(fmt.Sprintf("nft insert rule %s %s %s counter jump %s", tableFamily, internetTable, preroutingChain, whitelistPrerouting), nil)
	if err != nil {
		return err
	}

	// Authenticated-device bypass: a device with an active session (in
	// connected_macs_set) skips the captive-portal DNAT. This is global (not
	// per-interface), so it is set up once here instead of per-interface. It is
	// appended AFTER the whitelist jump and BEFORE the DNAT rule (which
	// SetCaptivePortalTarget appends later), giving the correct precedence:
	// whitelist → authenticated → portal-IP bypass → DNAT everyone else.
	err = cmd.Exec(fmt.Sprintf("nft add rule %s %s %s ether saddr @%s counter accept", tableFamily, internetTable, preroutingChain, connMacSet), nil)
	if err != nil {
		return err
	}

	// Portal-IP bypass: traffic from a captive interface already destined to a
	// portal-serving IP (interfaces.json) is accepted here, BEFORE the DNAT rule
	// SetCaptivePortalTarget appends later, so it is never rewritten to the main
	// portal IP. Without this, a client sent to its own gateway's portal (e.g.
	// http://20.0.0.1 by RedirectToLanIP) would have the destination silently
	// swapped to the main IP while its Host header still names its own gateway.
	err = cmd.Exec(fmt.Sprintf("nft add rule %s %s %s iifname @%s ip daddr @%s counter accept", tableFamily, internetTable, preroutingChain, captiveIfacesSet, portalIpsV4Set), nil)
	if err != nil {
		return err
	}
	err = cmd.Exec(fmt.Sprintf("nft add rule %s %s %s iifname @%s ip6 daddr @%s counter accept", tableFamily, internetTable, preroutingChain, captiveIfacesSet, portalIpsV6Set), nil)
	if err != nil {
		return err
	}

	// Start the background reconciler that keeps whitelisted MACs' return-traffic
	// IPs current, independent of session events.
	startWhitelistReconciler()

	return nil
}

// SetInterfaceMode reconciles a LAN device's membership in the managed_ifaces
// and captive_ifaces sets:
//   - managed → session firewall + anti-tethering apply (managed_ifaces).
//   - captive → the port-80 portal redirect also applies (captive_ifaces).
//     Only meaningful together with managed; captive is always a subset.
//   - neither → the interface is untouched (the transparency passthrough accepts
//     its traffic and no custom rule references it).
// Membership changes take effect on the next packet, so an admin toggling on the
// Interfaces page applies instantly with no rule surgery. Best-effort idempotent.
func SetInterfaceMode(dev string, managed bool, captive bool) error {
	if dev == "" {
		return fmt.Errorf("interface device is required")
	}

	contextInfo := fmt.Sprintf("Device=%s, Managed=%t, Captive=%t", dev, managed, captive)

	_, err := nftQue.ExecWithTimeout(
		10*time.Second,
		"Set Interface Managed Mode",
		contextInfo,
		func() (any, error) {
			setMembership(managedIfacesSet, dev, managed)
			setMembership(captiveIfacesSet, dev, captive)
			return nil, nil
		},
	)
	return err
}

// setMembership adds or removes an interface device from an nftables ifname set.
// The ifname is quoted — device names can contain dots (e.g. "br-lan.22").
// Best-effort: a missing element on delete (or a duplicate on add) is not fatal.
func setMembership(set, dev string, member bool) {
	if member {
		cmd.Exec(fmt.Sprintf("nft add element %s %s %s '{ \"%s\" }' 2>/dev/null || true", tableFamily, internetTable, set, dev), nil)
	} else {
		cmd.Exec(fmt.Sprintf("nft delete element %s %s %s '{ \"%s\" }' 2>/dev/null || true", tableFamily, internetTable, set, dev), nil)
	}
}

// SetPortalIPs reconciles the portal_ips_v4/v6 sets with the portal-serving
// addresses resolved from interfaces.json: the gateway IP of every
// captive-portal-enabled LAN plus the main portal interface's IP. Traffic from
// captive interfaces to these addresses bypasses the port-80 DNAT and the
// forward-chain session gate (rules installed once in Setup), so a client on
// one captive subnet can always reach the portal hosted on another's address.
// The sets are flushed and repopulated, so it is idempotent and safe to call on
// every ApplyPortalConfig. Unparseable entries are skipped; each IP lands in
// the set matching its family, so callers pass one mixed list.
func SetPortalIPs(ips []string) error {
	contextInfo := fmt.Sprintf("PortalIPs=%v", ips)

	_, err := nftQue.ExecWithTimeout(
		10*time.Second,
		"Set Portal IPs",
		contextInfo,
		func() (any, error) {
			cmd.Exec(fmt.Sprintf("nft flush set %s %s %s 2>/dev/null || true", tableFamily, internetTable, portalIpsV4Set), nil)
			cmd.Exec(fmt.Sprintf("nft flush set %s %s %s 2>/dev/null || true", tableFamily, internetTable, portalIpsV6Set), nil)
			for _, ip := range ips {
				if net.ParseIP(ip) == nil {
					continue
				}
				set := portalIpsV4Set
				if isIPv6(ip) {
					set = portalIpsV6Set
				}
				cmd.Exec(fmt.Sprintf("nft add element %s %s %s '{ %s }' 2>/dev/null || true", tableFamily, internetTable, set, ip), nil)
			}
			return nil, nil
		},
	)
	return err
}

// SetCaptivePortalTarget installs the single port-80 DNAT rule shared by every
// captive interface (matched by the captive_ifaces set), redirecting to the MAIN
// interface's IP. Because all captive interfaces DNAT to the same target, one
// rule covers them all — clients on a secondary subnet (e.g. 20.0.0.0/20) are
// redirected to the main portal IP (e.g. 10.0.0.1), which is a local router
// address and so is delivered via the INPUT hook (bypassing the forward-chain
// drop). It is re-runnable: the previous DNAT rule(s) are removed first, so
// calling it again after the main interface's IP changes just swaps the target.
// routerIp4 == "" removes the DNAT entirely (no captive redirect). Link-local
// IPv6 is rejected (nftables DNAT cannot target a link-local scope).
func SetCaptivePortalTarget(routerIp4 string, routerIp6 string) (err error) {
	if routerIp6 != "" {
		parsed := net.ParseIP(routerIp6)
		if parsed == nil || parsed.IsLinkLocalUnicast() {
			routerIp6 = ""
		}
	}

	contextInfo := fmt.Sprintf("RouterIPv4=%s, RouterIPv6=%s", routerIp4, routerIp6)

	_, err = nftQue.ExecWithTimeout(
		30*time.Second,
		"Set Captive Portal Target",
		contextInfo,
		func() (any, error) {
			// Remove any existing captive DNAT rule(s) first so the target can be
			// swapped when the main interface (or its IP) changes.
			deleteCaptiveDnatRules()

			var batch strings.Builder

			// Redirect plain HTTP (port 80) to the main portal IP (IPv4).
			// Port 443 is intentionally NOT intercepted: MITM'ing TLS breaks the
			// browser. Modern OSes are instead pointed at the portal via the RFC
			// 8910 advertisement (DHCP option 114); port 80 stays as the legacy
			// detection fallback for clients that still probe over HTTP.
			if routerIp4 != "" {
				batch.WriteString(fmt.Sprintf("add rule %s %s %s iifname @%s tcp dport { 80 } counter dnat ip to %s\n", tableFamily, internetTable, preroutingChain, captiveIfacesSet, routerIp4))
			}
			if routerIp6 != "" {
				batch.WriteString(fmt.Sprintf("add rule %s %s %s iifname @%s tcp dport { 80 } counter dnat ip6 to %s\n", tableFamily, internetTable, preroutingChain, captiveIfacesSet, routerIp6))
			}

			if batch.Len() == 0 {
				return nil, nil
			}

			nftCmd := fmt.Sprintf("nft -f - <<'EOF'\n%sEOF", batch.String())
			return nil, cmd.Exec(nftCmd, nil)
		},
	)
	return err
}

// deleteCaptiveDnatRules removes the captive-portal DNAT rule(s) from the
// prerouting chain by handle. The prerouting base chain only ever carries the
// whitelist jump, the authenticated-device bypass, and these DNAT rules, so any
// rule containing a dnat verdict is one of ours. Best-effort.
func deleteCaptiveDnatRules() {
	var out bytes.Buffer
	if err := cmd.ExecOutput(fmt.Sprintf("nft -j -a list chain %s %s %s", tableFamily, internetTable, preroutingChain), &out); err != nil {
		return
	}

	var result nftRuleList
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return
	}

	for _, entry := range result.Nftables {
		if entry.Rule == nil {
			continue
		}
		if ruleContainsDnat(entry.Rule.Expr) {
			cmd.Exec(fmt.Sprintf("nft delete rule %s %s %s handle %d 2>/dev/null || true", tableFamily, internetTable, preroutingChain, entry.Rule.Handle), nil)
		}
	}
}

// ruleContainsDnat reports whether a rule's expressions include a dnat verdict.
func ruleContainsDnat(exprs []interface{}) bool {
	for _, expr := range exprs {
		exprMap, ok := expr.(map[string]interface{})
		if !ok {
			continue
		}
		if _, ok := exprMap["dnat"]; ok {
			return true
		}
	}
	return false
}

func Connect(ip string, mac string) error {
	contextInfo := fmt.Sprintf("IP=%s, MAC=%s", ip, mac)

	_, err := nftQue.ExecWithTimeout(
		30*time.Second,
		"Connect Device",
		contextInfo,
		func() (any, error) {
			err := doConnect(ip, mac)
			return nil, err
		},
	)
	return err
}

func Disconnect(ip string, mac string) error {
	contextInfo := fmt.Sprintf("IP=%s, MAC=%s", ip, mac)

	_, err := nftQue.ExecWithTimeout(
		30*time.Second,
		"Disconnect Device",
		contextInfo,
		func() (any, error) {
			err := doDisconnect(ip, mac)
			return nil, err
		},
	)
	return err
}

func IsConnected(mac string) bool {
	contextInfo := fmt.Sprintf("MAC=%s", mac)

	result, err := nftQue.ExecWithTimeout(
		30*time.Second,
		"Check Connection Status",
		contextInfo,
		func() (any, error) {
			return isConnected(mac), nil
		},
	)

	if err != nil {
		return false
	}

	return result.(bool)
}

// IsWhitelisted reports whether mac has standing internet access granted via
// AllowMAC (the whitelist bypass), independent of any session. It reads the
// in-memory whitelist map directly under nftMu (no nft shell-out), so it is
// cheap enough for the hot captive-probe path. The MAC is normalized first, so
// callers may pass any case/separator form.
func IsWhitelisted(mac string) bool {
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		return false
	}
	nftMu.RLock()
	defer nftMu.RUnlock()
	return whitelistedMacs[normalizedMAC]
}

// GetMacByIp returns the normalized uppercase MAC address for a currently
// connected IP address (IPv4 or IPv6), or an empty string if the IP is not
// in the cache. The cache is populated on Connect and evicted on Disconnect,
// so it only contains entries for devices actively allowed through the firewall.
func GetMacByIp(ip string) string {
	nftMu.RLock()
	defer nftMu.RUnlock()
	return ipToMac[ip]
}

// GetMacsByIps returns a map of IP→MAC for a batch of IP addresses, acquiring
// the lock only once. This is more efficient than calling GetMacByIp in a loop.
// IPs not found in the cache are omitted from the result map.
func GetMacsByIps(ips []string) map[string]string {
	nftMu.RLock()
	defer nftMu.RUnlock()

	result := make(map[string]string, len(ips))
	for _, ip := range ips {
		if mac := ipToMac[ip]; mac != "" {
			result[ip] = mac
		}
	}
	return result
}

// isIPv6 returns true if ip is a valid IPv6 address (not IPv4 or IPv4-mapped).
func isIPv6(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.To4() == nil
}

func isConnected(mac string) bool {
	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		return false
	}

	err = cmd.Exec(fmt.Sprintf("nft get element %s %s %s '{ %s }'", tableFamily, internetTable, connMacSet, normalizedMAC), nil)
	return err == nil
}

func doConnect(ip string, mac string) error {
	// Validate IP address
	if _, err := sdkutils.ValidateIPAddress(ip); err != nil {
		return fmt.Errorf("invalid IP address: %v", err)
	}

	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %v", err)
	}

	// Choose the correct IP verdict map based on IP version (used for download accounting only).
	ipMap := connIpMap
	if isIPv6(ip) {
		ipMap = connIp6Map
	}

	// Step 1: Add this IP to the download-accounting verdict map.
	// Idempotent via || true — safe if called twice for the same IP.
	cmd.Exec(fmt.Sprintf("nft add element %s %s %s '{ %s : accept }' 2>/dev/null || true", tableFamily, internetTable, ipMap, ip), nil)

	// Step 2: Add MAC to the upload-accounting verdict map and the allow set.
	// These are idempotent — the second call for a dual-stack device is a no-op.
	cmd.Exec(fmt.Sprintf("nft add element %s %s %s '{ %s : accept }' 2>/dev/null || true", tableFamily, internetTable, connMacMap, normalizedMAC), nil)
	cmd.Exec(fmt.Sprintf("nft add element %s %s %s '{ %s }' 2>/dev/null || true", tableFamily, internetTable, connMacSet, normalizedMAC), nil)

	// Record IP→MAC and MAC→IPs mappings for traffic accounting and GetMacByIp().
	nftMu.Lock()
	ipToMac[ip] = normalizedMAC
	if macToIps[normalizedMAC] == nil {
		macToIps[normalizedMAC] = make(map[string]bool)
	}
	macToIps[normalizedMAC][ip] = true
	whitelisted := whitelistedMacs[normalizedMAC]
	nftMu.Unlock()

	// If this MAC is whitelisted (AllowMAC), learn the IP it just connected with
	// so return traffic works, and prune only a stale same-family IP from a prior
	// address. This is add-only — it never revokes access (that is DisallowMAC's job)
	// — so it stays independent of session lifecycle while still converging on the
	// device's current IP across a change.
	if whitelisted {
		syncWhitelistIPForMac(normalizedMAC, ip)
	}

	return nil
}

func AllowMAC(mac string) error {
	contextInfo := fmt.Sprintf("MAC=%s", mac)

	_, err := nftQue.ExecWithTimeout(
		30*time.Second,
		"Allow MAC (Whitelist)",
		contextInfo,
		func() (any, error) {
			return nil, doAllowMAC(mac)
		},
	)
	return err
}

// DisallowMAC revokes a whitelist grant created by AllowMAC — it removes the MAC
// from the whitelist bypass and clears the return-traffic IPs tracked for it. It
// does NOT hard-block the device: if the device still has an active session it
// keeps internet through the session path. For an absolute deny use BlockMAC.
func DisallowMAC(mac string) error {
	contextInfo := fmt.Sprintf("MAC=%s", mac)

	_, err := nftQue.ExecWithTimeout(
		30*time.Second,
		"Disallow MAC (Whitelist)",
		contextInfo,
		func() (any, error) {
			return nil, doDisallowMAC(mac)
		},
	)
	return err
}

// BlockMAC absolutely denies internet to a MAC regardless of session or whitelist
// state. It adds the MAC to a drop set evaluated at the TOP of the forward chain
// (above the session and whitelist accepts), drops the device's current
// download IP(s), and flushes conntrack so established connections cut instantly.
// Reverse with UnblockMAC.
func BlockMAC(mac string) error {
	contextInfo := fmt.Sprintf("MAC=%s", mac)

	_, err := nftQue.ExecWithTimeout(
		30*time.Second,
		"Block MAC (Hard Deny)",
		contextInfo,
		func() (any, error) {
			return nil, doBlockMAC(mac)
		},
	)
	return err
}

// UnblockMAC removes a hard block created by BlockMAC, restoring the device to
// whatever access it would otherwise have (session and/or whitelist). It does not
// grant any access on its own.
func UnblockMAC(mac string) error {
	contextInfo := fmt.Sprintf("MAC=%s", mac)

	_, err := nftQue.ExecWithTimeout(
		30*time.Second,
		"Unblock MAC (Hard Deny)",
		contextInfo,
		func() (any, error) {
			return nil, doUnblockMAC(mac)
		},
	)
	return err
}

func doAllowMAC(mac string) error {
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %v", err)
	}

	// Upload: accept client -> internet by source MAC.
	if err := cmd.Exec(fmt.Sprintf("nft add element %s %s %s '{ %s }' 2>/dev/null || true", tableFamily, internetTable, whitelistMacSet, normalizedMAC), nil); err != nil {
		return err
	}

	// Mark whitelisted BEFORE registering IPs so a concurrent Connect also syncs.
	nftMu.Lock()
	whitelistedMacs[normalizedMAC] = true
	nftMu.Unlock()

	// Download/return: accept internet -> client by destination IP. AllowMAC by
	// itself only covers the upload direction; without the client's current IP
	// here, replies fall through to the forward chain's policy drop. Resolve the
	// device's live IP(s) and register them now. If the device isn't connected
	// yet (no IP), doConnect learns it when it comes up; a later IP change is
	// converged by the same same-family-replace path (here and in doConnect).
	//
	// Then flush each IP's conntrack so the grant applies to in-flight
	// connections, not just new ones: a device that hit port 80 BEFORE being
	// whitelisted was captive-portal DNAT'd, and conntrack pins that flow to the
	// portal until it expires. Register the IP first, then flush, so the flow's
	// next packet re-evaluates against the now-permissive ruleset.
	for _, ip := range resolveMacIPs(normalizedMAC) {
		syncWhitelistIPForMac(normalizedMAC, ip)
		flushConntrackForIP(ip)
	}

	return nil
}

func doDisallowMAC(mac string) error {
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %v", err)
	}

	// Stop tracking and snapshot the IPs we registered for this MAC.
	nftMu.Lock()
	delete(whitelistedMacs, normalizedMAC)
	ips := whitelistMacIps[normalizedMAC]
	delete(whitelistMacIps, normalizedMAC)
	nftMu.Unlock()

	// Remove the upload bypass.
	cmd.Exec(fmt.Sprintf("nft delete element %s %s %s '{ %s }' 2>/dev/null || true", tableFamily, internetTable, whitelistMacSet, normalizedMAC), nil)

	// Remove every return-traffic IP we added for this MAC.
	for ip := range ips {
		removeWhitelistIP(ip)
	}

	return nil
}

func doBlockMAC(mac string) error {
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %v", err)
	}

	// Upload: drop client -> internet by source MAC. This element feeds the drop
	// rule at the top of the forward chain, so it beats both the session accept
	// (connected_macs_map) and the whitelist accept (whitelist_macs).
	if err := cmd.Exec(fmt.Sprintf("nft add element %s %s %s '{ %s }' 2>/dev/null || true", tableFamily, internetTable, blockedMacSet, normalizedMAC), nil); err != nil {
		return err
	}

	nftMu.Lock()
	blockedMacs[normalizedMAC] = true
	nftMu.Unlock()

	// Download/return: drop internet -> client by destination IP for the device's
	// current IP(s), and flush conntrack so established connections are cut now.
	// The MAC upload-drop above is the permanent, IP-independent guarantee (a
	// blocked client can't send, so it can't use the internet even if its IP later
	// changes); these IP drops + the flush just make the cutoff instant.
	for _, ip := range resolveMacIPs(normalizedMAC) {
		addBlockedIPForMac(normalizedMAC, ip)
		flushConntrackForIP(ip)
	}

	return nil
}

func doUnblockMAC(mac string) error {
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %v", err)
	}

	// Stop tracking and snapshot the download IPs we dropped for this MAC.
	nftMu.Lock()
	delete(blockedMacs, normalizedMAC)
	ips := blockedMacIps[normalizedMAC]
	delete(blockedMacIps, normalizedMAC)
	nftMu.Unlock()

	// Remove the upload deny.
	cmd.Exec(fmt.Sprintf("nft delete element %s %s %s '{ %s }' 2>/dev/null || true", tableFamily, internetTable, blockedMacSet, normalizedMAC), nil)

	// Remove every download deny we added for this MAC.
	for ip := range ips {
		removeBlockedIP(ip)
	}

	return nil
}

func doDisconnect(ip string, mac string) error {
	// Validate IP address
	if _, err := sdkutils.ValidateIPAddress(ip); err != nil {
		return fmt.Errorf("invalid IP address: %v", err)
	}

	// Validate and normalize MAC address
	normalizedMAC, err := sdkutils.ValidateAndNormalizeMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %v", err)
	}

	// Choose the correct IP verdict map based on IP version.
	ipMap := connIpMap
	if isIPv6(ip) {
		ipMap = connIp6Map
	}

	// Step 1: Remove this IP from the download-accounting verdict map.
	// Best-effort — if it was never added (e.g. partial connect failure), swallow.
	cmd.Exec(fmt.Sprintf("nft delete element %s %s %s '{ %s : accept }' 2>/dev/null || true", tableFamily, internetTable, ipMap, ip), nil)

	// Flush this IP's conntrack entries so the cutoff is immediate — established
	// flows otherwise survive until they time out naturally. Done per IP because
	// conntrack tuples are L3 (no MAC field).
	flushConntrackForIP(ip)

	// NOTE: a session disconnect intentionally does NOT touch the whitelist sets.
	// Whitelisting (AllowMAC) is independent of session state — a whitelisted
	// client that ends its session while still connected must keep internet
	// access. Whitelist return-traffic IPs are pruned only on an IP change
	// (same-family replace in doConnect/doAllowMAC) or revoked wholesale by
	// DisallowMAC.

	// Step 2: Update in-memory maps.  Only remove MAC-level entries when all
	// IPs for this device have been disconnected (handles dual-stack correctly).
	nftMu.Lock()
	delete(ipToMac, ip)
	remainingIPs := 0
	if ips, ok := macToIps[normalizedMAC]; ok {
		delete(ips, ip)
		remainingIPs = len(ips)
		if remainingIPs == 0 {
			delete(macToIps, normalizedMAC)
		}
	}
	nftMu.Unlock()

	// Step 3: Once all IPs for this MAC are gone, remove the MAC from the
	// nftables allow set/map and flush conntrack entries so existing connections
	// are cut immediately rather than left alive until they time out naturally.
	if remainingIPs == 0 {
		cmd.Exec(fmt.Sprintf("nft delete element %s %s %s '{ %s : accept }' 2>/dev/null || true", tableFamily, internetTable, connMacMap, normalizedMAC), nil)
		cmd.Exec(fmt.Sprintf("nft delete element %s %s %s '{ %s }' 2>/dev/null || true", tableFamily, internetTable, connMacSet, normalizedMAC), nil)
	}

	return nil
}

// =============================================================================
// WHITELIST RETURN-TRAFFIC HELPERS (internal)
//
// The whitelist_forward chain accepts upload by source MAC (whitelist_macs) and
// download/return by destination client IP (whitelist_client_ips_v4/v6). AllowMAC
// fills the MAC set; these helpers manage the client-IP sets so a whitelisted
// device gets working bidirectional internet, and stay correct as its IP changes.
// =============================================================================

// resolveMacIPs returns the best-known current IP(s) for a MAC: the in-memory
// connected set (covers IPv4+IPv6 for session-managed devices) unioned with the
// kernel ARP table (covers IPv4 for any device with a neighbor entry). Reads the
// ARP table once. Used by AllowMAC to register return traffic immediately, even
// before a Connect fires.
func resolveMacIPs(normalizedMAC string) []string {
	return resolveMacIPsFrom(normalizedMAC, arpReverseTable())
}

// resolveMacIPsFrom is resolveMacIPs against an already-read ARP reverse index,
// so a batch caller (the reconciler) reads /proc/net/arp once per pass instead
// of once per MAC.
func resolveMacIPsFrom(normalizedMAC string, arpByMac map[string][]string) []string {
	seen := make(map[string]bool)
	var ips []string

	nftMu.RLock()
	for ip := range macToIps[normalizedMAC] {
		if !seen[ip] {
			seen[ip] = true
			ips = append(ips, ip)
		}
	}
	nftMu.RUnlock()

	for _, ip := range arpByMac[normalizedMAC] {
		if ip != "" && !seen[ip] {
			seen[ip] = true
			ips = append(ips, ip)
		}
	}

	return ips
}

// arpReverseTable reads /proc/net/arp once and returns a reverse index of
// normalized MAC → IPv4 address(es). MACs are normalized through the same
// validator as everywhere else so keys match the normalizedMAC lookups exactly;
// unparseable/incomplete entries (e.g. all-zero MACs) are skipped.
func arpReverseTable() map[string][]string {
	rev := make(map[string][]string)
	for ip, mac := range arp.Table() {
		norm, err := sdkutils.ValidateAndNormalizeMAC(mac)
		if err != nil {
			continue
		}
		rev[norm] = append(rev[norm], ip)
	}
	return rev
}

// startWhitelistReconciler launches the background reconcile loop exactly once.
func startWhitelistReconciler() {
	whitelistReconcilerOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(whitelistReconcileInterval)
			defer ticker.Stop()
			for range ticker.C {
				// Serialize the pass with all other firewall ops so it cannot race
				// AllowMAC/DisallowMAC/Connect/BlockMAC on the maps or nft sets.
				nftQue.ExecWithTimeout(
					30*time.Second,
					"Reconcile Whitelist IPs",
					"",
					func() (any, error) {
						reconcileWhitelistIPs()
						return nil, nil
					},
				)
			}
		}()
	})
}

// reconcileWhitelistIPs re-resolves each whitelisted MAC's current IP(s) from the
// in-memory cache and the ARP table and syncs them into the return-traffic sets.
// This catches IP changes that never produced a Connect event. It only ever adds
// the current IP and prunes a stale same-family one — a device that is currently
// offline (no resolvable IP) keeps its last-known entry, preserving the
// session-independent whitelist guarantee until DisallowMAC.
func reconcileWhitelistIPs() {
	nftMu.RLock()
	macs := make([]string, 0, len(whitelistedMacs))
	for mac := range whitelistedMacs {
		macs = append(macs, mac)
	}
	nftMu.RUnlock()

	if len(macs) == 0 {
		return
	}

	// Read /proc/net/arp once for the whole pass, not once per MAC.
	arpByMac := arpReverseTable()

	for _, mac := range macs {
		for _, ip := range resolveMacIPsFrom(mac, arpByMac) {
			syncWhitelistIPForMac(mac, ip)
		}
	}
}

// syncWhitelistIPForMac registers ip for a whitelisted MAC's return traffic and
// prunes any previously-registered IP of the SAME family for that MAC (a stale
// address left over from before an IP change). It is add-only with respect to
// access: it never removes a different-family IP (dual-stack stays intact) and is
// the ONLY converging path — session disconnect deliberately does not prune — so
// a whitelisted device keeps internet across session teardown and IP changes.
func syncWhitelistIPForMac(normalizedMAC, ip string) {
	if ip == "" {
		return
	}
	newIsV6 := isIPv6(ip)

	nftMu.RLock()
	var stale []string
	for old := range whitelistMacIps[normalizedMAC] {
		if old != ip && isIPv6(old) == newIsV6 {
			stale = append(stale, old)
		}
	}
	nftMu.RUnlock()

	for _, old := range stale {
		removeWhitelistIPForMac(normalizedMAC, old)
	}
	addWhitelistIPForMac(normalizedMAC, ip)
}

// whitelistSetForIP returns the whitelist client-IP set matching the IP's family.
func whitelistSetForIP(ip string) string {
	if isIPv6(ip) {
		return whitelistIpsV6Set
	}
	return whitelistIpsV4Set
}

// addWhitelistIPForMac registers a client IP for return traffic and records it
// under the MAC so DisallowMAC/Disconnect can remove exactly what was added.
// Idempotent at both the nftables and tracking-map level.
func addWhitelistIPForMac(normalizedMAC, ip string) {
	if ip == "" {
		return
	}
	cmd.Exec(fmt.Sprintf("nft add element %s %s %s '{ %s }' 2>/dev/null || true", tableFamily, internetTable, whitelistSetForIP(ip), ip), nil)

	nftMu.Lock()
	if whitelistMacIps[normalizedMAC] == nil {
		whitelistMacIps[normalizedMAC] = make(map[string]bool)
	}
	whitelistMacIps[normalizedMAC][ip] = true
	nftMu.Unlock()
}

// removeWhitelistIPForMac removes a single client IP's return-traffic entry and
// untracks it for the MAC.
func removeWhitelistIPForMac(normalizedMAC, ip string) {
	removeWhitelistIP(ip)

	nftMu.Lock()
	if ips := whitelistMacIps[normalizedMAC]; ips != nil {
		delete(ips, ip)
		if len(ips) == 0 {
			delete(whitelistMacIps, normalizedMAC)
		}
	}
	nftMu.Unlock()
}

// removeWhitelistIP removes a client IP from whichever whitelist client-IP set it
// belongs to (best-effort — a missing element is not an error).
func removeWhitelistIP(ip string) {
	if ip == "" {
		return
	}
	cmd.Exec(fmt.Sprintf("nft delete element %s %s %s '{ %s }' 2>/dev/null || true", tableFamily, internetTable, whitelistSetForIP(ip), ip), nil)
}

// =============================================================================
// HARD-BLOCK HELPERS (internal)
//
// The forward chain drops by source MAC (blocked_macs) and by destination client
// IP (blocked_client_ips_v4/v6) ABOVE every accept rule. BlockMAC fills the MAC
// set (the permanent guarantee); these helpers manage the client-IP sets so the
// download direction is cut at block time too.
// =============================================================================

// blockedSetForIP returns the blocked client-IP set matching the IP's family.
func blockedSetForIP(ip string) string {
	if isIPv6(ip) {
		return blockedIpsV6Set
	}
	return blockedIpsV4Set
}

// addBlockedIPForMac drops a client IP's return traffic and records it under the
// MAC so UnblockMAC can remove exactly what was added. Idempotent.
func addBlockedIPForMac(normalizedMAC, ip string) {
	if ip == "" {
		return
	}
	cmd.Exec(fmt.Sprintf("nft add element %s %s %s '{ %s }' 2>/dev/null || true", tableFamily, internetTable, blockedSetForIP(ip), ip), nil)

	nftMu.Lock()
	if blockedMacIps[normalizedMAC] == nil {
		blockedMacIps[normalizedMAC] = make(map[string]bool)
	}
	blockedMacIps[normalizedMAC][ip] = true
	nftMu.Unlock()
}

// removeBlockedIP removes a client IP from whichever blocked client-IP set it
// belongs to (best-effort — a missing element is not an error).
func removeBlockedIP(ip string) {
	if ip == "" {
		return
	}
	cmd.Exec(fmt.Sprintf("nft delete element %s %s %s '{ %s }' 2>/dev/null || true", tableFamily, internetTable, blockedSetForIP(ip), ip), nil)
}

// flushConntrackForIP deletes conntrack entries originating from a client IP so
// a firewall change (grant via AllowMAC, revoke via Disconnect) takes effect on
// already-established connections instead of only new ones. We flush by IP, not
// MAC: conntrack tuples are L3/L4 and carry no MAC field, so a MAC filter would
// match nothing. Best-effort — conntrack may be absent on some OpenWRT images,
// so errors are swallowed.
func flushConntrackForIP(ip string) {
	if ip == "" {
		return
	}
	cmd.Exec(fmt.Sprintf("conntrack -D --orig-src %s 2>/dev/null || true", ip), nil)
}
