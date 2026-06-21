package api

import (
	"net/http"
	"sync"

	sdkapi "sdk/api"
)

func NewPaymentMgr() *PaymentsMgr {
	return &PaymentsMgr{
		handlers: make(map[string]sdkapi.PurchaseExecuteHandler),
	}
}

type PaymentsMgr struct {
	providers []*PaymentProvider

	// handlers maps a callback plugin package to its in-process execute handler.
	// Purchase.Execute() looks up the handler by the purchase's callback plugin
	// and calls it directly.
	mu       sync.RWMutex
	handlers map[string]sdkapi.PurchaseExecuteHandler
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

// registerExecuteHandler stores the in-process execute handler for a callback
// plugin package. The last registration for a given package wins.
func (self *PaymentsMgr) registerExecuteHandler(pkg string, handler sdkapi.PurchaseExecuteHandler) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.handlers[pkg] = handler
}

// findExecuteHandler returns the handler registered for the given callback
// plugin package, if any.
func (self *PaymentsMgr) findExecuteHandler(pkg string) (sdkapi.PurchaseExecuteHandler, bool) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	handler, ok := self.handlers[pkg]
	return handler, ok
}
