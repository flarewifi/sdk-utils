package plugins

import (
	connmgr "sdk/api/connmgr"
	payments "sdk/api/payments"
	plugin "sdk/api/plugin"
)

func NewPaymentMgr() *PaymentsMgr {
	return &PaymentsMgr{}
}

type PaymentsMgr struct {
	providers []*PaymentProvider
}

func (self *PaymentsMgr) Options(clnt connmgr.IClientDevice) []PaymentOption {
	opts := []PaymentOption{}
	for _, prvdr := range self.providers {
		for _, opt := range prvdr.PaymentOpts(clnt) {
			opts = append(opts, NewPaymentOpt(prvdr.api, opt))
		}
	}
	return opts
}

func (self *PaymentsMgr) FindByUuid(clnt connmgr.IClientDevice, uuid string) (PaymentOption, bool) {
	methods := self.Options(clnt)
	for _, opt := range methods {
		if opt.UUID == uuid {
			return opt, true
		}
	}
	return PaymentOption{}, false
}

func (self *PaymentsMgr) NewPaymentProvider(api plugin.IPluginApi, provider payments.IPaymentProvider) {
	prvdr := NewPaymentProvider(api, provider)
	self.providers = append(self.providers, prvdr)
}
