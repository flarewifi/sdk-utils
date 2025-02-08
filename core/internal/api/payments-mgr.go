package api

import (
	"net/http"
	sdkapi "sdk/api"
)

func NewPaymentMgr() *PaymentsMgr {
	return &PaymentsMgr{}
}

type PaymentsMgr struct {
	providers []*PaymentProvider
}

func (self *PaymentsMgr) AllOptions(r *http.Request) []*PaymentOption {
	opts := []*PaymentOption{}
	for _, prvdr := range self.providers {
		for _, opt := range prvdr.GetOptions(r) {
			opts = append(opts, NewPaymentOpt(prvdr.api, opt))
		}
	}

	return opts
}

func (self *PaymentsMgr) FindOption(r *http.Request, uuid string) (*PaymentOption, bool) {
	options := self.AllOptions(r)
	for _, opt := range options {
		if opt.UUID() == uuid {
			return opt, true
		}
	}
	return nil, false
}

func (self *PaymentsMgr) NewPaymentProvider(api sdkapi.IPluginApi, provider sdkapi.IPaymentProvider) {
	prvdr := NewPaymentProvider(api, provider)
	self.providers = append(self.providers, prvdr)
}
