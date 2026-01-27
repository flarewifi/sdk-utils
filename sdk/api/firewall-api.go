package sdkapi

// OpenIpForClientDeviceParams defines parameters for opening firewall access for a client device to a destination IP
type OpenIpForClientDeviceParams struct {
	DstIp       string // Dst IP address to allow access to
	IpAddr      string // Client device IP address (for return traffic filtering)
	MacAddr     string // Client device MAC address (for source traffic filtering)
	TimeoutSecs int    // Timeout in seconds (0 = permanent, >0 = auto-remove after timeout)
}

// CloseIpForClientDeviceParams defines parameters for closing firewall access for a client device
type CloseIpForClientDeviceParams struct {
	DstIp   string // Dst IP address to close access to
	MacAddr string // Client device MAC address
}

type DstIpGroupClient struct {
	MacAddr string // Client device MAC address
	IpAddr  string // Client device IP address (for return traffic filtering)
}

type IFirewallAPI interface {

	// ResolveHostnameToIps takes a hostname as input and returns a list of associated IP addresses.
	ResolveHostnameToIps(hostname string) ([]string, error)

	// OpenIpForClientDevice opens firewall access for a specific client device (identified by MAC address)
	// to a destination IP address. All ports are opened for bidirectional traffic.
	// If TimeoutSecs is 0, the rule is permanent. If TimeoutSecs > 0, the rule is automatically removed after the specified duration.
	OpenIpForClientDevice(params OpenIpForClientDeviceParams) error

	// CloseIpForClientDevice removes firewall access for a specific client device to a destination IP address.
	CloseIpForClientDevice(params CloseIpForClientDeviceParams) error

	// CreateDstIpGroup creates a named group of destination IP addresses that can be used for firewall rules.
	CreateDstIpGroup(name string, ips ...string) error

	// AddIpsToDstIpGroup adds IP addresses to an existing named destination IP group.
	AddIpsToDstIpGroup(name string, ips ...string) error

	// ChangeDstIpGroup replaces the IP addresses in an existing named destination IP group with a new set of IP addresses.
	ChangeDstIpGroup(name string, ips ...string) error

	// DstIpGroupExists checks if a named destination IP group exists.
	DstIpGroupExists(name string) (bool, error)

	// AllowClientDeviceToDstIpGroup allows a specific client device to access all IPs in a named destination IP group.
	AllowClientDeviceToDstIpGroup(clnt DstIpGroupClient, groupName string, timeoutSecs int) error

	// RemoveClientDeviceFromDstIpGroup removes access for a specific client device to all IPs in a named destination IP group.
	RemoveClientDeviceFromDstIpGroup(clnt DstIpGroupClient, groupName string) error
}
