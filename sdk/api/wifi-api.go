package sdkapi

type WifiClientEvent string

var (
	WifiEventClientConnected    WifiClientEvent = "wifi:client:connected"
	WifiEventClientDisconnected WifiClientEvent = "wifi:client:disconnected"
)

// WifiEvent represents a WiFi client event with interface info
type WifiEvent struct {
	Interface string          // WiFi interface name (e.g., "wlan0")
	Mac       string          // Client MAC address
	Event     WifiClientEvent // Event type
}

type IWifiApi interface {
	// Listen returns a channel that receives all WiFi client events
	Listen() <-chan WifiEvent

	// OnWifiClientEvent registers a callback for specific WiFi client events
	OnWifiClientEvent(event WifiClientEvent, callback func(client IWifiClient))
}

type IWifiClient interface {
	MacAddress() string
}
