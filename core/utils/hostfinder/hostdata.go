package hostfinder

// HostData is the network identity of a client host resolved from the machine's
// DHCP lease and ARP/NDP tables. This is core's internal type; the SDK exposes
// the same shape to plugins as sdkapi.RequestHost (GetRequestHost converts).
type HostData struct {
	MacAddr  string
	IpAddr   string
	Hostname string
}
