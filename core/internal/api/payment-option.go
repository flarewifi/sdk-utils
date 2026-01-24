package api

import (
	sdkapi "sdk/api"
)

func NewPaymentOpt(api sdkapi.IPluginApi, opt sdkapi.PaymentOption) *PaymentOption {
	// No more UUID generation - use plugin-provided UUID
	return &PaymentOption{api, opt}
}

type PaymentOption struct {
	api        sdkapi.IPluginApi
	paymentOpt sdkapi.PaymentOption
}

func (self *PaymentOption) UUID() string {
	return self.paymentOpt.UUID
}

func (self *PaymentOption) Name() string {
	return self.paymentOpt.Name
}

func (self *PaymentOption) Label() string {
	return self.paymentOpt.Name
}

func (self *PaymentOption) URL() string {
	params := []string{}
	for k, v := range self.paymentOpt.RouteParams {
		params = append(params, k, v)
	}

	routeURL := self.api.Http().Helpers().UrlForRoute(self.paymentOpt.RouteName, params...)
	return routeURL
}
