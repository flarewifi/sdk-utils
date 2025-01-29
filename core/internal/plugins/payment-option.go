package plugins

import (
	sdkapi "sdk/api"
)

func NewPaymentOpt(api sdkapi.IPluginApi, opt sdkapi.PaymentOption) PaymentOption {
	uuid := api.Info().Package + "::" + opt.Name
	return PaymentOption{api, opt, uuid}
}

type PaymentOption struct {
	api  sdkapi.IPluginApi
	Opt  sdkapi.PaymentOption
	UUID string
}
