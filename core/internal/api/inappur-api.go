package api

import (
	"net/http"
	sdkapi "sdk/api"
)

func NewInAppPurchaseApi(plugin *PluginApi) {
	purApi := &InAppPurchaseApi{plugin}
	plugin.InAppPurchaseAPI = purApi
}

type InAppPurchaseApi struct {
	plugin *PluginApi
}

func (self *InAppPurchaseApi) VerifyPurchase(sdkapi.InAppCheckoutItem) error {
	return nil
}

func (self *InAppPurchaseApi) VerifySubscription(sdkapi.InAppSubscriptionItem) error {
	return nil
}

func (self *InAppPurchaseApi) PurchaseGuardMiddleware(sdkapi.InAppCheckoutItem) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}

func (self *InAppPurchaseApi) SubscriptionGuardMiddleware(sdkapi.InAppSubscriptionItem) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
