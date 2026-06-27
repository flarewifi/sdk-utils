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

func (self *InAppPurchaseApi) CheckOneTimePurchase(productID string) (sdkapi.InAppOneTimePurchaseStatus, error) {
	return sdkapi.InAppOneTimePurchaseStatus{}, nil
}

func (self *InAppPurchaseApi) CheckSubscription(planID string) (sdkapi.InAppSubscriptionStatus, error) {
	return sdkapi.InAppSubscriptionStatus{}, nil
}

func (self *InAppPurchaseApi) PurchaseGuardMiddleware(sdkapi.InAppOneTimePurchaseStatus) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}

func (self *InAppPurchaseApi) SubscriptionGuardMiddleware(sdkapi.InAppSubscription) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
