package plugins

import (
	connmgr "sdk/api/connmgr"
	payments "sdk/api/payments"
	plugin "sdk/api/plugin"
)


func NewPaymentProvider(api plugin.IPluginApi, provider payments.IPaymentProvider) *PaymentProvider {
    prv := &PaymentProvider{api, provider}
    return prv
}

type PaymentProvider struct {
	api      plugin.IPluginApi
	provider payments.IPaymentProvider
}

func (self *PaymentProvider) IProvider() payments.IPaymentProvider {
	return self.provider
}

func (self *PaymentProvider) PaymentOpts(clnt connmgr.IClientDevice) []payments.PaymentOpt {
	return self.provider.PaymentOpts(clnt)
}
