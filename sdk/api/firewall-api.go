package sdkapi

type DstIpGroupClient struct {
	MacAddr string // Client device MAC address
	IpAddr  string // Client device IP address (for return traffic filtering)
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

	// AllowClientToDstIpGroup allows a specific client device to access all IPs in a named destination IP group.
	AllowClientToDstIpGroup(clnt DstIpGroupClient, groupName string, timeoutSecs int) error

	// RemoveClientFromDstIpGroup removes access for a specific client device to all IPs in a named destination IP group.
	RemoveClientFromDstIpGroup(clnt DstIpGroupClient, groupName string) error
}
