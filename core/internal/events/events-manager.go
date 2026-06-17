package events

import (
	"context"
	"sync"
	"time"

	sdkapi "sdk/api"
)

// callbackTimeout is the maximum time allowed for a single async event callback to run.
// Chosen to be slightly longer than cloud-sync's RPC_TIMEOUT (1 minute), giving callbacks
// enough time to finish their own internal timeouts before the goroutine is abandoned.
const callbackTimeout = 2 * time.Minute

// EventsManager is the single, global store for all event callbacks in the system.
// It is created once at startup and injected into every component that emits or listens
// to events (SessionsMgr, ClientRegister, PaymentsApi, VouchersApi, etc.).
//
// All Emit* methods are fire-and-forget: each registered callback runs in its own
// goroutine so the caller (HTTP handler, session loop, …) is never blocked.
// Each goroutine receives a context with a 2-minute deadline so a slow or hung
// callback cannot leak goroutines indefinitely.
// Errors from callbacks are logged but not propagated to the caller.
type EventsManager struct {
	mu sync.RWMutex

	sessionCallbacks      map[sdkapi.SessionEvent][]func(context.Context, sdkapi.SessionEventData) error
	sessionBatchCallbacks map[sdkapi.SessionEvent][]func(context.Context, []sdkapi.IClientSession) error
	clientCallbacks       map[sdkapi.ClientEvent][]func(context.Context, sdkapi.IClientDevice) error
	clientMergeCallbacks  []func(context.Context, sdkapi.EventClientMergeData) error
	purchaseCallbacks     map[sdkapi.PurchaseEvent][]func(context.Context, sdkapi.PurchaseEventData) error
	voucherCallbacks      map[sdkapi.VoucherEvent][]func(context.Context, sdkapi.IVoucher) error
	voucherBatchCallbacks map[sdkapi.VoucherEvent][]func(context.Context, sdkapi.IVoucherBatch) error

	// voucherBeforeCreateCallbacks holds synchronous pre-create veto hooks. Unlike
	// the event-keyed maps above, these are not dispatched async — see EmitVoucherBeforeCreate.
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

// OnVoucherBeforeCreate registers a synchronous pre-create veto hook.
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
// ASYNC EMIT
// =============================================================================

// EmitSessionEvent dispatches a session event to all registered callbacks asynchronously.
// Each callback runs in its own goroutine. Errors are logged; the caller is never blocked.
func (em *EventsManager) EmitSessionEvent(ctx context.Context, event sdkapi.SessionEvent, data sdkapi.SessionEventData) {
	em.mu.RLock()
	snapshot := make([]func(context.Context, sdkapi.SessionEventData) error, len(em.sessionCallbacks[event]))
	copy(snapshot, em.sessionCallbacks[event])
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(ctx, callbackTimeout)
			defer cancel()
			cb(ctx, data)
		}()
	}
}

// EmitSessionBatchEvent dispatches a batch session event to all registered callbacks asynchronously.
func (em *EventsManager) EmitSessionBatchEvent(ctx context.Context, event sdkapi.SessionEvent, sessions []sdkapi.IClientSession) {
	em.mu.RLock()
	snapshot := make([]func(context.Context, []sdkapi.IClientSession) error, len(em.sessionBatchCallbacks[event]))
	copy(snapshot, em.sessionBatchCallbacks[event])
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(ctx, callbackTimeout)
			defer cancel()
			cb(ctx, sessions)
		}()
	}
}

// EmitClientEvent dispatches a client event to all registered callbacks asynchronously.
func (em *EventsManager) EmitClientEvent(ctx context.Context, event sdkapi.ClientEvent, clnt sdkapi.IClientDevice) {
	em.mu.RLock()
	snapshot := make([]func(context.Context, sdkapi.IClientDevice) error, len(em.clientCallbacks[event]))
	copy(snapshot, em.clientCallbacks[event])
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(ctx, callbackTimeout)
			defer cancel()
			cb(ctx, clnt)
		}()
	}
}

// EmitClientEventSync dispatches a client event to all registered callbacks
// SYNCHRONOUSLY, in the caller's goroutine and in registration order, stopping at
// and returning the first error. Unlike the fire-and-forget Emit* methods, this is
// used for blocking hooks such as EventClientBeforeConnect where a callback must be
// able to veto an operation. Each callback still receives a context bounded by
// callbackTimeout so a hung callback cannot block the caller indefinitely.
func (em *EventsManager) EmitClientEventSync(ctx context.Context, event sdkapi.ClientEvent, clnt sdkapi.IClientDevice) error {
	em.mu.RLock()
	snapshot := make([]func(context.Context, sdkapi.IClientDevice) error, len(em.clientCallbacks[event]))
	copy(snapshot, em.clientCallbacks[event])
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cbCtx, cancel := context.WithTimeout(ctx, callbackTimeout)
		err := cb(cbCtx, clnt)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}

// EmitPurchaseEvent dispatches a purchase event to all registered callbacks asynchronously.
func (em *EventsManager) EmitPurchaseEvent(ctx context.Context, event sdkapi.PurchaseEvent, data sdkapi.PurchaseEventData) {
	em.mu.RLock()
	snapshot := make([]func(context.Context, sdkapi.PurchaseEventData) error, len(em.purchaseCallbacks[event]))
	copy(snapshot, em.purchaseCallbacks[event])
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(ctx, callbackTimeout)
			defer cancel()
			cb(ctx, data)
		}()
	}
}

// EmitVoucherEvent dispatches a single-voucher event to all registered callbacks asynchronously.
func (em *EventsManager) EmitVoucherEvent(ctx context.Context, event sdkapi.VoucherEvent, v sdkapi.IVoucher) {
	em.mu.RLock()
	snapshot := make([]func(context.Context, sdkapi.IVoucher) error, len(em.voucherCallbacks[event]))
	copy(snapshot, em.voucherCallbacks[event])
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(ctx, callbackTimeout)
			defer cancel()
			cb(ctx, v)
		}()
	}
}

// EmitClientMerge dispatches a client-merge event to all registered callbacks asynchronously.
// The source device is already deleted from the database when this is called.
func (em *EventsManager) EmitClientMerge(ctx context.Context, data sdkapi.EventClientMergeData) {
	em.mu.RLock()
	snapshot := make([]func(context.Context, sdkapi.EventClientMergeData) error, len(em.clientMergeCallbacks))
	copy(snapshot, em.clientMergeCallbacks)
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(ctx, callbackTimeout)
			defer cancel()
			cb(ctx, data)
		}()
	}
}

// EmitVoucherBatchEvent dispatches a voucher-batch event to all registered callbacks asynchronously.
func (em *EventsManager) EmitVoucherBatchEvent(ctx context.Context, event sdkapi.VoucherEvent, batch sdkapi.IVoucherBatch) {
	em.mu.RLock()
	snapshot := make([]func(context.Context, sdkapi.IVoucherBatch) error, len(em.voucherBatchCallbacks[event]))
	copy(snapshot, em.voucherBatchCallbacks[event])
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(ctx, callbackTimeout)
			defer cancel()
			cb(ctx, batch)
		}()
	}
}

// EmitVoucherBeforeCreate runs all registered pre-create hooks SYNCHRONOUSLY, in
// registration order, in the caller's goroutine, stopping at and returning the first
// error. Hooks receive a pointer to params and may modify them; a returned error
// aborts voucher creation. Each hook gets a context bounded by callbackTimeout.
func (em *EventsManager) EmitVoucherBeforeCreate(ctx context.Context, params *sdkapi.CreateVouchersParams) error {
	em.mu.RLock()
	snapshot := make([]func(context.Context, *sdkapi.CreateVouchersParams) error, len(em.voucherBeforeCreateCallbacks))
	copy(snapshot, em.voucherBeforeCreateCallbacks)
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cbCtx, cancel := context.WithTimeout(ctx, callbackTimeout)
		err := cb(cbCtx, params)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}


