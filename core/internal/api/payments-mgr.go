package api

import (
	"net/http"
	"sync"

	sdkapi "sdk/api"
)

func NewPaymentMgr() *PaymentsMgr {
	return &PaymentsMgr{
		purchaseEventCallbacks: sync.Map{},
	}
}

type PaymentsMgr struct {
	providers              []*PaymentProvider
	purchaseEventCallbacks sync.Map // map[sdkapi.PurchaseEvent][]func(data sdkapi.PurchaseEventData) error
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

// OnPurchaseEvent registers a callback for purchase events.
func (self *PaymentsMgr) OnPurchaseEvent(event sdkapi.PurchaseEvent, callback func(data sdkapi.PurchaseEventData) error) {
	callbacks := []func(data sdkapi.PurchaseEventData) error{}
	if existing, ok := self.purchaseEventCallbacks.Load(event); ok {
		callbacks = existing.([]func(data sdkapi.PurchaseEventData) error)
	}
	callbacks = append(callbacks, callback)
	self.purchaseEventCallbacks.Store(event, callbacks)
}

// EmitPurchaseEvent emits a purchase event to all registered callbacks.
func (self *PaymentsMgr) EmitPurchaseEvent(event sdkapi.PurchaseEvent, data sdkapi.PurchaseEventData) error {
	if callbacksVal, exists := self.purchaseEventCallbacks.Load(event); exists {
		callbacks := callbacksVal.([]func(data sdkapi.PurchaseEventData) error)
		for _, callback := range callbacks {
			if err := callback(data); err != nil {
				return err
			}
		}
	}
	return nil
}
