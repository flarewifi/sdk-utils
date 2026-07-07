package sdkapi

type DstIpGroupClient struct {
	MacAddr  string // Client device MAC address
	IpAddr   string // Primary IP address – kept for backward compatibility (prefer Ipv4Addr/Ipv6Addr)
	Ipv4Addr string // IPv4 address for return traffic filtering (empty if device is IPv6-only)
	Ipv6Addr string // IPv6 address for return traffic filtering (empty if device is IPv4-only)
}

type IFirewallAPI interface {

	// ResolveHostnameToIps takes a hostname as input and returns a list of associated IP addresses.
	ResolveHostnameToIps(hostname string) ([]string, error)

	// CreateDstIpGroup creates a named group of destination IP addresses that can be used for firewall rules.
	CreateDstIpGroup(name string, ips ...string) error

	// AddIpsToDstIpGroup adds IP addresses to an existing named destination IP group.
	AddIpsToDstIpGroup(name string, ips ...string) error

	// ChangeDstIpGroup replaces the IP addresses in an existing named destination IP group with a new set of IP addresses.
	ChangeDstIpGroup(name string, ips ...string) error

	// DstIpGroupExists checks if a named destination IP group exists.
	DstIpGroupExists(name string) (bool, error)

	// DeleteDstIpGroup removes a named destination IP group and all its nftables infrastructure.
	// All clients currently allowed access through this group will immediately lose access.
	// Cancels any scheduled automatic removals for clients in this group.
	DeleteDstIpGroup(name string) error

	// AllowClientToDstIpGroup allows a specific client device to access all IPs in a named destination IP group.
	AllowClientToDstIpGroup(clnt DstIpGroupClient, groupName string, timeoutSecs int) error

	// RemoveClientFromDstIpGroup removes access for a specific client device to all IPs in a named destination IP group.
	RemoveClientFromDstIpGroup(clnt DstIpGroupClient, groupName string) error

	// CreateServicePort creates a named service port definition that allows traffic on specified protocol(s) and port.
	// Name is slugified for safe use in nftables identifiers.
	// Protocols must contain at least one protocol ("tcp" or "udp").
	// Port must be a valid port number (1-65535).
	// Optional dstIPs restrict traffic to specific destination IPs (empty = any destination).
	CreateServicePort(name string, protocols []string, port int, dstIPs ...string) error

	// ServicePortExists checks if a named service port exists.
	ServicePortExists(name string) (bool, error)

	// DeleteServicePort removes a named service port and all its nftables infrastructure.
	// All clients currently allowed access through this service port will immediately lose access.
	// Cancels any scheduled automatic removals for clients using this service port.
	DeleteServicePort(name string) error

	// AllowClientToServicePort allows a specific client device to access a named service port.
	// If timeoutSecs > 0, access is automatically revoked after the specified duration.
	// If timeoutSecs <= 0, access persists until explicitly removed or the firewall is reset.
	AllowClientToServicePort(clnt DstIpGroupClient, servicePortName string, timeoutSecs int) error

	// RemoveClientFromServicePort removes access for a specific client device from a named service port.
	RemoveClientFromServicePort(clnt DstIpGroupClient, servicePortName string) error

	// AllowMAC opens the firewall for a MAC address, bypassing the captive portal.
	// It grants working bidirectional internet on its own (it resolves and tracks
	// the device's IP for return traffic internally). This is ephemeral — the
	// caller is responsible for persisting the whitelist and re-applying on boot.
	AllowMAC(mac string) error

	// DisallowMAC revokes an AllowMAC grant — it removes the MAC from the whitelist
	// bypass and clears the return-traffic IPs tracked for it. It is NOT a block:
	// a device that still has an active session keeps internet through the session.
	// Use BlockMAC for an absolute deny.
	DisallowMAC(mac string) error

	// BlockMAC absolutely denies internet access to a MAC, regardless of whether
	// the device has an active session or is whitelisted. The deny is evaluated
	// before all accept rules, the device's in-flight connections are cut
	// immediately, and the block persists until UnblockMAC (ephemeral across
	// reboots — re-apply on boot if it must survive a restart).
	BlockMAC(mac string) error

	// UnblockMAC removes a BlockMAC hard block, restoring whatever access the
	// device would otherwise have (session and/or whitelist). It grants nothing on
	// its own.
	UnblockMAC(mac string) error

	// AddPreRoutingChainBeforeInternet creates chainName (if it doesn't already
	// exist) in the shared firewall table and wires a jump to it from the very
	// top of the prerouting chain — before the whitelist/session bypass and the
	// captive-portal DNAT. The caller owns every rule inside chainName (added via
	// its own nft calls); this only creates the chain and registers the jump.
	// MUST be called from an api.Network().OnReady() callback, not from Init() —
	// the shared table doesn't exist until nftables setup has completed.
	// Idempotent.
	AddPreRoutingChainBeforeInternet(chainName string) error

	// AddPreRoutingChainAfterInternet is the same as
	// AddPreRoutingChainBeforeInternet but wires the jump at the end of the
	// prerouting rules the core sets up — a captive-portal DNAT rule added later
	// (WiFi hotspot) can still land after this jump. Idempotent; same OnReady
	// timing requirement.
	AddPreRoutingChainAfterInternet(chainName string) error

	// AddForwardChainBeforeInternet creates chainName (if it doesn't already
	// exist) and wires a jump to it from the very top of the forward chain —
	// before even the hard-block rules. A terminal verdict inside chainName
	// short-circuits the built-in forward logic for matching packets; anything
	// chainName doesn't resolve falls through to the built-in rules as normal.
	// Idempotent; same OnReady timing requirement as
	// AddPreRoutingChainBeforeInternet.
	AddForwardChainBeforeInternet(chainName string) error

	// AddForwardChainAfterInternet is the same as AddForwardChainBeforeInternet
	// but wires the jump at the end of the forward chain — after the built-in
	// hard-block/whitelist/session rules but before the chain's own drop policy.
	// Idempotent; same OnReady timing requirement.
	AddForwardChainAfterInternet(chainName string) error
}
