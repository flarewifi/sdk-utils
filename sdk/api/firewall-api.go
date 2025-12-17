package sdkapi

// OpenIpForClientDeviceParams defines parameters for opening firewall access for a client device to a destination IP
type OpenIpForClientDeviceParams struct {
	DestinationIp string // Destination IP address to allow access to
	IpAddr        string // Client device IP address (for return traffic filtering)
	MacAddr       string // Client device MAC address (for source traffic filtering)
	TimeoutSecs   int    // Timeout in seconds (0 = permanent, >0 = auto-remove after timeout)
}

// CloseIpForClientDeviceParams defines parameters for closing firewall access for a client device
type CloseIpForClientDeviceParams struct {
	DestinationIp string // Destination IP address to close access to
	MacAddr       string // Client device MAC address
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
}
