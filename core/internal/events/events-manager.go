package events

import (
	"context"
	"sync"
	"time"

	sdkapi "sdk/api"
)

// callbackTimeout bounds the context passed to a single event callback. Because
// callbacks now run SYNCHRONOUSLY in the emitter's goroutine, this deadline cannot
// forcibly preempt a blocking callback — it only signals cancellation to callbacks
// that honor their context. A handler that needs to run longer, or must not block
// the caller at all, should spawn its own goroutine inside the callback.
const callbackTimeout = 2 * time.Minute

// EventsManager is the single, global store for all event callbacks in the system.
// It is created once at startup and injected into every component that emits or listens
// to events (SessionsMgr, ClientRegister, PaymentsApi, VouchersApi, etc.).
//
// All Emit* methods dispatch SYNCHRONOUSLY: registered callbacks run sequentially in
// the caller's goroutine, in registration order. Every Emit* method returns the first
// non-nil error produced by a callback. All callbacks still run even when one fails,
// so a single faulty handler never suppresses the others. Callers that can cancel an
// operation (e.g. SessionsMgr.Connect, VouchersApi.CreateVouchers) check this error
// and abort; notification callers ignore it.
//
// Synchronous dispatch makes async an explicit, opt-in decision of the handler: a
// callback that must not block the caller (or that runs longer than callbackTimeout)
// should spawn its own goroutine. Each callback receives a context bounded by
// callbackTimeout so a cooperative callback can bail out early.
type EventsManager struct {
	mu sync.RWMutex

	sessionCallbacks      map[sdkapi.SessionEvent][]func(context.Context, sdkapi.SessionEventData) error
	sessionBatchCallbacks map[sdkapi.SessionEvent][]func(context.Context, []sdkapi.IClientSession) error
	clientCallbacks       map[sdkapi.ClientEvent][]func(context.Context, sdkapi.IClientDevice) error
	clientMergeCallbacks  []func(context.Context, sdkapi.EventClientMergeData) error
	purchaseCallbacks     map[sdkapi.PurchaseEvent][]func(context.Context, sdkapi.PurchaseEventData) error
	voucherCallbacks      map[sdkapi.VoucherEvent][]func(context.Context, sdkapi.IVoucher) error
	voucherBatchCallbacks map[sdkapi.VoucherEvent][]func(context.Context, sdkapi.IVoucherBatch) error

	// voucherBeforeCreateCallbacks holds the pre-create hooks. They use the same
	// synchronous dispatch as every other event; a returned error cancels voucher
	// creation (see EmitVoucherBeforeCreate).
	voucherBeforeCreateCallbacks []func(context.Context, *sdkapi.CreateVouchersParams) error
}

// NewEventsManager constructs an EventsManager ready for use.
func NewEventsManager() *EventsManager {
	return &EventsManager{
		sessionCallbacks:      make(map[sdkapi.SessionEvent][]func(context.Context, sdkapi.SessionEventData) error),
		sessionBatchCallbacks: make(map[sdkapi.SessionEvent][]func(context.Context, []sdkapi.IClientSession) error),
		clientCallbacks:       make(map[sdkapi.ClientEvent][]func(context.Context, sdkapi.IClientDevice) error),
		purchaseCallbacks:     make(map[sdkapi.PurchaseEvent][]func(context.Context, sdkapi.PurchaseEventData) error),
		voucherCallbacks:      make(map[sdkapi.VoucherEvent][]func(context.Context, sdkapi.IVoucher) error),
		voucherBatchCallbacks: make(map[sdkapi.VoucherEvent][]func(context.Context, sdkapi.IVoucherBatch) error),
	}
}

// =============================================================================
// REGISTRATION
// =============================================================================

// OnSessionEvent registers a callback that fires whenever the given session event occurs.
func (em *EventsManager) OnSessionEvent(event sdkapi.SessionEvent, cb func(context.Context, sdkapi.SessionEventData) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.sessionCallbacks[event] = append(em.sessionCallbacks[event], cb)
}

// OnSessionBatchEvent registers a callback that fires whenever a batch session event occurs.
func (em *EventsManager) OnSessionBatchEvent(event sdkapi.SessionEvent, cb func(context.Context, []sdkapi.IClientSession) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.sessionBatchCallbacks[event] = append(em.sessionBatchCallbacks[event], cb)
}

// OnClientEvent registers a callback that fires whenever the given client event occurs.
func (em *EventsManager) OnClientEvent(event sdkapi.ClientEvent, cb func(context.Context, sdkapi.IClientDevice) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.clientCallbacks[event] = append(em.clientCallbacks[event], cb)
}

// OnPurchaseEvent registers a callback that fires whenever the given purchase event occurs.
func (em *EventsManager) OnPurchaseEvent(event sdkapi.PurchaseEvent, cb func(context.Context, sdkapi.PurchaseEventData) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.purchaseCallbacks[event] = append(em.purchaseCallbacks[event], cb)
}

// OnVoucherEvent registers a callback that fires whenever the given single-voucher event occurs.
func (em *EventsManager) OnVoucherEvent(event sdkapi.VoucherEvent, cb func(context.Context, sdkapi.IVoucher) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.voucherCallbacks[event] = append(em.voucherCallbacks[event], cb)
}

// OnVoucherBatchEvent registers a callback that fires whenever the given voucher-batch event occurs.
func (em *EventsManager) OnVoucherBatchEvent(event sdkapi.VoucherEvent, cb func(context.Context, sdkapi.IVoucherBatch) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.voucherBatchCallbacks[event] = append(em.voucherBatchCallbacks[event], cb)
}

// OnVoucherBeforeCreate registers a pre-create hook. The hook runs synchronously
// before a voucher batch is created; returning an error cancels creation.
func (em *EventsManager) OnVoucherBeforeCreate(cb func(context.Context, *sdkapi.CreateVouchersParams) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.voucherBeforeCreateCallbacks = append(em.voucherBeforeCreateCallbacks, cb)
}

// OnClientMerge registers a callback that fires after two device records have been
// successfully merged. The source device is deleted before callbacks are invoked.
func (em *EventsManager) OnClientMerge(cb func(context.Context, sdkapi.EventClientMergeData) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.clientMergeCallbacks = append(em.clientMergeCallbacks, cb)
}

// =============================================================================
// SYNCHRONOUS EMIT
// =============================================================================

// EmitSessionEvent dispatches a session event to all registered callbacks synchronously.
func (em *EventsManager) EmitSessionEvent(ctx context.Context, event sdkapi.SessionEvent, data sdkapi.SessionEventData) error {
	em.mu.RLock()
	cbs := append([]func(context.Context, sdkapi.SessionEventData) error(nil), em.sessionCallbacks[event]...)
	em.mu.RUnlock()
	return dispatch(ctx, cbs, data)
}

// EmitSessionBatchEvent dispatches a batch session event to all registered callbacks synchronously.
func (em *EventsManager) EmitSessionBatchEvent(ctx context.Context, event sdkapi.SessionEvent, sessions []sdkapi.IClientSession) error {
	em.mu.RLock()
	cbs := append([]func(context.Context, []sdkapi.IClientSession) error(nil), em.sessionBatchCallbacks[event]...)
	em.mu.RUnlock()
	return dispatch(ctx, cbs, sessions)
}

// EmitClientEvent dispatches a client event to all registered callbacks synchronously.
// A returned error lets callers that can cancel the operation (e.g.
// EventClientBeforeConnect from SessionsMgr.Connect) abort it.
func (em *EventsManager) EmitClientEvent(ctx context.Context, event sdkapi.ClientEvent, clnt sdkapi.IClientDevice) error {
	em.mu.RLock()
	cbs := append([]func(context.Context, sdkapi.IClientDevice) error(nil), em.clientCallbacks[event]...)
	em.mu.RUnlock()
	return dispatch(ctx, cbs, clnt)
}

// EmitPurchaseEvent dispatches a purchase event to all registered callbacks synchronously.
func (em *EventsManager) EmitPurchaseEvent(ctx context.Context, event sdkapi.PurchaseEvent, data sdkapi.PurchaseEventData) error {
	em.mu.RLock()
	cbs := append([]func(context.Context, sdkapi.PurchaseEventData) error(nil), em.purchaseCallbacks[event]...)
	em.mu.RUnlock()
	return dispatch(ctx, cbs, data)
}

// EmitVoucherEvent dispatches a single-voucher event to all registered callbacks synchronously.
func (em *EventsManager) EmitVoucherEvent(ctx context.Context, event sdkapi.VoucherEvent, v sdkapi.IVoucher) error {
	em.mu.RLock()
	cbs := append([]func(context.Context, sdkapi.IVoucher) error(nil), em.voucherCallbacks[event]...)
	em.mu.RUnlock()
	return dispatch(ctx, cbs, v)
}

// EmitVoucherBatchEvent dispatches a voucher-batch event to all registered callbacks synchronously.
func (em *EventsManager) EmitVoucherBatchEvent(ctx context.Context, event sdkapi.VoucherEvent, batch sdkapi.IVoucherBatch) error {
	em.mu.RLock()
	cbs := append([]func(context.Context, sdkapi.IVoucherBatch) error(nil), em.voucherBatchCallbacks[event]...)
	em.mu.RUnlock()
	return dispatch(ctx, cbs, batch)
}

// EmitClientMerge dispatches a client-merge event to all registered callbacks synchronously.
// The source device is already deleted from the database when this is called.
func (em *EventsManager) EmitClientMerge(ctx context.Context, data sdkapi.EventClientMergeData) error {
	em.mu.RLock()
	cbs := append([]func(context.Context, sdkapi.EventClientMergeData) error(nil), em.clientMergeCallbacks...)
	em.mu.RUnlock()
	return dispatch(ctx, cbs, data)
}

// EmitVoucherBeforeCreate runs all registered pre-create hooks synchronously, in
// registration order. Hooks receive a pointer to params and may modify them; a
// returned error cancels voucher creation. Because hooks may run even after one of
// them cancels creation, a pre-create hook must be a side-effect-free check.
func (em *EventsManager) EmitVoucherBeforeCreate(ctx context.Context, params *sdkapi.CreateVouchersParams) error {
	em.mu.RLock()
	cbs := append([]func(context.Context, *sdkapi.CreateVouchersParams) error(nil), em.voucherBeforeCreateCallbacks...)
	em.mu.RUnlock()
	return dispatch(ctx, cbs, params)
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// dispatch runs every callback synchronously, in order, in the caller's goroutine.
// All callbacks run even if some return errors; the first non-nil error is returned
// so callers that can cancel the operation may do so. Each callback receives a
// context bounded by callbackTimeout.
func dispatch[T any](ctx context.Context, cbs []func(context.Context, T) error, arg T) error {
	var firstErr error
	for _, cb := range cbs {
		cbCtx, cancel := context.WithTimeout(ctx, callbackTimeout)
		err := cb(cbCtx, arg)
		cancel()
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
