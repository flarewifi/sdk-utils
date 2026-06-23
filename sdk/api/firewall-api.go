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
	// This is ephemeral — the caller is responsible for persistence.
	// For return traffic, the caller must also manage client IP sets separately.
	AllowMAC(mac string) error

	// BlockMAC closes the firewall for a previously allowed MAC address.
	// Does not flush associated client IPs — the caller handles that.
	BlockMAC(mac string) error
}
