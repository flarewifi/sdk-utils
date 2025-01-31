package plugins

import (
	"net/http"
	sdkapi "sdk/api"
)

func NewPaymentProvider(api sdkapi.IPluginApi, provider sdkapi.IPaymentProvider) *PaymentProvider {
	prv := &PaymentProvider{api, provider}
	return prv
}

type PaymentProvider struct {
	api      sdkapi.IPluginApi
	provider sdkapi.IPaymentProvider
}

func (self *PaymentProvider) IProvider() sdkapi.IPaymentProvider {
	return self.provider
}

func (self *PaymentProvider) PaymentOpts(r *http.Request) []sdkapi.PaymentOption {
	return self.provider.PaymentOpts(r)
}
