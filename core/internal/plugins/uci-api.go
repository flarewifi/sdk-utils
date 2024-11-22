package plugins

import (
	"core/internal/utils/uci"
	sdkuci "sdk/api/uci"

	gouci "github.com/digineo/go-uci"
)

type UciApi struct {
	networkApi  *uci.UciNetworkApi
	dhcpApi     *uci.UciDhcpApi
	wirelessApi *uci.UciWirelessApi
}

func NewUciApi(pluginApi *PluginApi) {
	uciApi := &UciApi{}
	pluginApi.UciAPI = uciApi
}

func (self *UciApi) Network() sdkuci.INetworkApi {
	return self.networkApi
}

func (self *UciApi) Dhcp() sdkuci.IDhcpApi {
	return self.dhcpApi
}

func (self *UciApi) Wireless() sdkuci.IWirelessApi {
	return self.wirelessApi
}

func (self *UciApi) Uci() gouci.Tree {
	return uci.UciTree
}
