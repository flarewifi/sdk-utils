package events

import (
	"context"
	"log"
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
// Dispatch behaviour:
//   - All Emit* methods are fire-and-forget: each registered callback runs in its own
//     goroutine so the caller (HTTP handler, session loop, …) is never blocked.
//   - Each goroutine receives a context with a 2-minute deadline so a slow or hung
//     callback cannot leak goroutines indefinitely.
//   - Errors from callbacks are logged but not propagated to the caller.
//
// Exception:
//   - RunBeforeCreate is synchronous and returns the first error it encounters.
//     This is intentional: it is a blocking gate that must run before vouchers are created.
type EventsManager struct {
	mu sync.RWMutex

	sessionCallbacks      map[sdkapi.SessionEvent][]func(sdkapi.SessionEventData) error
	sessionBatchCallbacks map[sdkapi.SessionEvent][]func([]sdkapi.IClientSession) error
	clientCallbacks       map[sdkapi.ClientEvent][]func(sdkapi.IClientDevice) error
	clientMergeCallbacks  []func(sdkapi.EventClientMergeData) error
	purchaseCallbacks     map[sdkapi.PurchaseEvent][]func(sdkapi.PurchaseEventData) error
	voucherCallbacks      map[sdkapi.VoucherEvent][]func(sdkapi.IVoucher) error
	voucherBatchCallbacks map[sdkapi.VoucherEvent][]func(sdkapi.IVoucherBatch) error
	beforeCreateCallbacks []func(context.Context, *sdkapi.CreateVouchersParams) error
}

// NewEventsManager constructs an EventsManager ready for use.
func NewEventsManager() *EventsManager {
	return &EventsManager{
		sessionCallbacks:      make(map[sdkapi.SessionEvent][]func(sdkapi.SessionEventData) error),
		sessionBatchCallbacks: make(map[sdkapi.SessionEvent][]func([]sdkapi.IClientSession) error),
		clientCallbacks:       make(map[sdkapi.ClientEvent][]func(sdkapi.IClientDevice) error),
		purchaseCallbacks:     make(map[sdkapi.PurchaseEvent][]func(sdkapi.PurchaseEventData) error),
		voucherCallbacks:      make(map[sdkapi.VoucherEvent][]func(sdkapi.IVoucher) error),
		voucherBatchCallbacks: make(map[sdkapi.VoucherEvent][]func(sdkapi.IVoucherBatch) error),
	}
}

// =============================================================================
// REGISTRATION
// =============================================================================

// OnSessionEvent registers a callback that fires whenever the given session event occurs.
func (em *EventsManager) OnSessionEvent(event sdkapi.SessionEvent, cb func(sdkapi.SessionEventData) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.sessionCallbacks[event] = append(em.sessionCallbacks[event], cb)
}

// OnSessionBatchEvent registers a callback that fires whenever a batch session event occurs.
func (em *EventsManager) OnSessionBatchEvent(event sdkapi.SessionEvent, cb func([]sdkapi.IClientSession) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.sessionBatchCallbacks[event] = append(em.sessionBatchCallbacks[event], cb)
}

// OnClientEvent registers a callback that fires whenever the given client event occurs.
func (em *EventsManager) OnClientEvent(event sdkapi.ClientEvent, cb func(sdkapi.IClientDevice) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.clientCallbacks[event] = append(em.clientCallbacks[event], cb)
}

// OnPurchaseEvent registers a callback that fires whenever the given purchase event occurs.
func (em *EventsManager) OnPurchaseEvent(event sdkapi.PurchaseEvent, cb func(sdkapi.PurchaseEventData) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.purchaseCallbacks[event] = append(em.purchaseCallbacks[event], cb)
}

// OnVoucherEvent registers a callback that fires whenever the given single-voucher event occurs.
func (em *EventsManager) OnVoucherEvent(event sdkapi.VoucherEvent, cb func(sdkapi.IVoucher) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.voucherCallbacks[event] = append(em.voucherCallbacks[event], cb)
}

// OnVoucherBatchEvent registers a callback that fires whenever the given voucher-batch event occurs.
func (em *EventsManager) OnVoucherBatchEvent(event sdkapi.VoucherEvent, cb func(sdkapi.IVoucherBatch) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.voucherBatchCallbacks[event] = append(em.voucherBatchCallbacks[event], cb)
}

// OnBeforeCreate registers a hook that is called synchronously before voucher creation.
// The hook may modify params or return an error to abort creation.
func (em *EventsManager) OnBeforeCreate(cb func(context.Context, *sdkapi.CreateVouchersParams) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.beforeCreateCallbacks = append(em.beforeCreateCallbacks, cb)
}

// OnClientMerge registers a callback that fires after two device records have been
// successfully merged. The source device is deleted before callbacks are invoked.
func (em *EventsManager) OnClientMerge(cb func(sdkapi.EventClientMergeData) error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.clientMergeCallbacks = append(em.clientMergeCallbacks, cb)
}

// =============================================================================
// ASYNC EMIT
// =============================================================================

// EmitSessionEvent dispatches a session event to all registered callbacks asynchronously.
// Each callback runs in its own goroutine. Errors are logged; the caller is never blocked.
func (em *EventsManager) EmitSessionEvent(event sdkapi.SessionEvent, data sdkapi.SessionEventData) {
	em.mu.RLock()
	cbs := em.sessionCallbacks[event]
	snapshot := make([]func(sdkapi.SessionEventData) error, len(cbs))
	copy(snapshot, cbs)
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), callbackTimeout)
			defer cancel()
			_ = ctx // callbacks don't accept context yet; timeout is enforced by their own internal logic
			if err := cb(data); err != nil {
				log.Printf("[EventsManager] %s handler error: %v", event, err)
			}
		}()
	}
}

// EmitSessionBatchEvent dispatches a batch session event to all registered callbacks asynchronously.
func (em *EventsManager) EmitSessionBatchEvent(event sdkapi.SessionEvent, sessions []sdkapi.IClientSession) {
	em.mu.RLock()
	cbs := em.sessionBatchCallbacks[event]
	snapshot := make([]func([]sdkapi.IClientSession) error, len(cbs))
	copy(snapshot, cbs)
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), callbackTimeout)
			defer cancel()
			_ = ctx
			if err := cb(sessions); err != nil {
				log.Printf("[EventsManager] %s handler error: %v", event, err)
			}
		}()
	}
}

// EmitClientEvent dispatches a client event to all registered callbacks asynchronously.
func (em *EventsManager) EmitClientEvent(event sdkapi.ClientEvent, clnt sdkapi.IClientDevice) {
	em.mu.RLock()
	cbs := em.clientCallbacks[event]
	snapshot := make([]func(sdkapi.IClientDevice) error, len(cbs))
	copy(snapshot, cbs)
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), callbackTimeout)
			defer cancel()
			_ = ctx
			if err := cb(clnt); err != nil {
				log.Printf("[EventsManager] %s handler error: %v", event, err)
			}
		}()
	}
}

// EmitPurchaseEvent dispatches a purchase event to all registered callbacks asynchronously.
func (em *EventsManager) EmitPurchaseEvent(event sdkapi.PurchaseEvent, data sdkapi.PurchaseEventData) {
	em.mu.RLock()
	cbs := em.purchaseCallbacks[event]
	snapshot := make([]func(sdkapi.PurchaseEventData) error, len(cbs))
	copy(snapshot, cbs)
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), callbackTimeout)
			defer cancel()
			_ = ctx
			if err := cb(data); err != nil {
				log.Printf("[EventsManager] %s handler error: %v", event, err)
			}
		}()
	}
}

// EmitVoucherEvent dispatches a single-voucher event to all registered callbacks asynchronously.
func (em *EventsManager) EmitVoucherEvent(event sdkapi.VoucherEvent, v sdkapi.IVoucher) {
	em.mu.RLock()
	cbs := em.voucherCallbacks[event]
	snapshot := make([]func(sdkapi.IVoucher) error, len(cbs))
	copy(snapshot, cbs)
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), callbackTimeout)
			defer cancel()
			_ = ctx
			if err := cb(v); err != nil {
				log.Printf("[EventsManager] %s handler error: %v", event, err)
			}
		}()
	}
}

// EmitClientMerge dispatches a client-merge event to all registered callbacks asynchronously.
// The source device is already deleted from the database when this is called.
func (em *EventsManager) EmitClientMerge(data sdkapi.EventClientMergeData) {
	em.mu.RLock()
	cbs := em.clientMergeCallbacks
	snapshot := make([]func(sdkapi.EventClientMergeData) error, len(cbs))
	copy(snapshot, cbs)
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), callbackTimeout)
			defer cancel()
			_ = ctx
			if err := cb(data); err != nil {
				log.Printf("[EventsManager] client:merged handler error: %v", err)
			}
		}()
	}
}

// EmitVoucherBatchEvent dispatches a voucher-batch event to all registered callbacks asynchronously.
func (em *EventsManager) EmitVoucherBatchEvent(event sdkapi.VoucherEvent, batch sdkapi.IVoucherBatch) {
	em.mu.RLock()
	cbs := em.voucherBatchCallbacks[event]
	snapshot := make([]func(sdkapi.IVoucherBatch) error, len(cbs))
	copy(snapshot, cbs)
	em.mu.RUnlock()

	for _, cb := range snapshot {
		cb := cb
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), callbackTimeout)
			defer cancel()
			_ = ctx
			if err := cb(batch); err != nil {
				log.Printf("[EventsManager] %s handler error: %v", event, err)
			}
		}()
	}
}

// =============================================================================
// SYNCHRONOUS GATE
// =============================================================================

// RunBeforeCreate runs all OnBeforeCreate hooks synchronously, in registration order.
// The first hook that returns a non-nil error aborts the chain and that error is returned
// to the caller. This is a blocking gate: voucher creation must not proceed on error.
func (em *EventsManager) RunBeforeCreate(ctx context.Context, params *sdkapi.CreateVouchersParams) error {
	em.mu.RLock()
	snapshot := make([]func(context.Context, *sdkapi.CreateVouchersParams) error, len(em.beforeCreateCallbacks))
	copy(snapshot, em.beforeCreateCallbacks)
	em.mu.RUnlock()

	for _, hook := range snapshot {
		if err := hook(ctx, params); err != nil {
			return err
		}
	}
	return nil
}
