package plugins

import (
	payments "sdk/api/payments"
	plugin "sdk/api/plugin"
)

func NewPaymentOpt(api plugin.IPluginApi, opt payments.PaymentOpt) PaymentOption {
	uuid := api.Pkg() + "::" + opt.OptName
	return PaymentOption{api, opt, uuid}
}

type PaymentOption struct {
	api  plugin.IPluginApi
	Opt  payments.PaymentOpt
	UUID string
}
