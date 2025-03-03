package api

import (
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func NewPaymentOpt(api sdkapi.IPluginApi, opt sdkapi.PaymentOption) *PaymentOption {
	seed := api.Info().Package + "::" + opt.Name
	uuid := sdkutils.Sha1Hash(seed)
	return &PaymentOption{api, opt, uuid}
}

type PaymentOption struct {
	api        sdkapi.IPluginApi
	paymentOpt sdkapi.PaymentOption
	uuid       string
}

func (self *PaymentOption) UUID() string {
	return self.uuid
}

func (self *PaymentOption) Name() string {
	return self.paymentOpt.Name
}

func (self *PaymentOption) Label() string {
	return self.paymentOpt.Label
}

func (self *PaymentOption) URL() string {
	params := []string{}
	for k, v := range self.paymentOpt.RouteParams {
		params = append(params, k, v)
	}

	routeURL := self.api.Http().Helpers().UrlForRoute(self.paymentOpt.RouteName, params...)
	return routeURL
}
