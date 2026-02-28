package api

import (
	"log"
	"sync"

	"core/internal/modules/ubus"
	sdkapi "sdk/api"
)

// Global WiFi event handlers shared across all plugin instances
var (
	globalWifiMu       sync.Mutex
	globalWifiHandlers = make(map[sdkapi.WifiClientEvent][]func(sdkapi.IWifiClient))
)

func NewWifiApi(pluginApi *PluginApi, wifiMgr *ubus.WifiMgr) *WifiApi {
	api := &WifiApi{
		pluginApi: pluginApi,
		wifiMgr:   wifiMgr,
	}
	pluginApi.WifiAPI = api
	return api
}

// WifiApi implements sdkapi.IWifiApi
type WifiApi struct {
	pluginApi *PluginApi
	wifiMgr   *ubus.WifiMgr
}

// Listen returns a channel for receiving WiFi events
func (self *WifiApi) Listen() <-chan sdkapi.WifiEvent {
	ubusEventCh := self.wifiMgr.Listen()
	sdkEventCh := make(chan sdkapi.WifiEvent)

	go func() {
		for evt := range ubusEventCh {
			sdkEventCh <- sdkapi.WifiEvent{
				Interface: evt.Interface,
				Mac:       evt.Mac,
				Event:     evt.Event,
			}
		}
		close(sdkEventCh)
	}()

	return sdkEventCh
}

// OnWifiClientEvent registers a callback for WiFi client events.
// Handlers are registered globally to allow cross-plugin event delivery.
func (self *WifiApi) OnWifiClientEvent(event sdkapi.WifiClientEvent, callback func(client sdkapi.IWifiClient)) {
	globalWifiMu.Lock()
	defer globalWifiMu.Unlock()
	globalWifiHandlers[event] = append(globalWifiHandlers[event], callback)
	log.Printf("[WifiApi] Registered handler for event: %s (total handlers: %d)", event, len(globalWifiHandlers[event]))
}

// wifiClient implements sdkapi.IWifiClient
type wifiClient struct {
	macAddress string
}

func (c *wifiClient) MacAddress() string {
	return c.macAddress
}

// EmitWifiEvent emits a WiFi client event to all registered handlers
func EmitWifiEvent(event sdkapi.WifiClientEvent, mac string) {
	globalWifiMu.Lock()
	callbacks := globalWifiHandlers[event]
	callbacksCopy := make([]func(sdkapi.IWifiClient), len(callbacks))
	copy(callbacksCopy, callbacks)
	globalWifiMu.Unlock()

	log.Printf("[WifiApi] Emitting event %s for MAC %s to %d handlers", event, mac, len(callbacksCopy))

	client := &wifiClient{macAddress: mac}
	for i, cb := range callbacksCopy {
		log.Printf("[WifiApi] Invoking handler %d/%d for event %s", i+1, len(callbacksCopy), event)
		cb(client)
	}
}
