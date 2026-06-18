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

	// handlers maps "<callbackPkg>::<route>" to the in-process execute handler
	// registered by a callback plugin. It replaces the loopback HTTP webhook:
	// Purchase.Execute() looks up the handler here and calls it directly.
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
// plugin's webhook route. The last registration for a given key wins.
func (self *PaymentsMgr) registerExecuteHandler(pkg, route string, handler sdkapi.PurchaseExecuteHandler) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.handlers[executeHandlerKey(pkg, route)] = handler
}

// findExecuteHandler returns the handler registered for the given callback
// plugin package and route, if any.
func (self *PaymentsMgr) findExecuteHandler(pkg, route string) (sdkapi.PurchaseExecuteHandler, bool) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	handler, ok := self.handlers[executeHandlerKey(pkg, route)]
	return handler, ok
}

func executeHandlerKey(pkg, route string) string {
	return pkg + "::" + route
}
