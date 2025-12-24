package api

import (
	"core/internal/utils/uci"
	sdkapi "sdk/api"

	gouci "github.com/digineo/go-uci"
)

type UciApi struct {
	networkApi  *uci.UciNetworkApi
	dhcpApi     *uci.UciDhcpApi
	wirelessApi *uci.UciWirelessApi
}

func NewUciApi(pluginApi *PluginApi) {
	uciApi := &UciApi{
		networkApi:  uci.NewUciNetworkApi(),
		dhcpApi:     uci.NewUciDhcpApi(),
		wirelessApi: uci.NewUciWirelessApi(),
	}
	pluginApi.UciAPI = uciApi
}

func (self *UciApi) Network() sdkapi.IUciNetworkApi {
	return self.networkApi
}

func (self *UciApi) Dhcp() sdkapi.IDhcpApi {
	return self.dhcpApi
}

func (self *UciApi) Wireless() sdkapi.IWirelessApi {
	return self.wirelessApi
}

func (self *UciApi) Uci() gouci.Tree {
	return uci.UciTree
}
