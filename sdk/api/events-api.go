/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"context"
)

// IEventsApi is the unified event subscription API for plugins.
//
// All registration methods are safe to call concurrently.
//
// DISPATCH IS SYNCHRONOUS. When an event fires, its registered callbacks run
// sequentially in the emitter's goroutine, in registration order. A callback that
// must not block the emitting operation — or that runs long — should spawn its own
// goroutine: deciding sync vs async is the handler's responsibility, not the core's.
//
// NO DATABASE TRANSACTION IS EVER HELD OPEN WHILE CALLBACKS RUN. Every core operation
// that emits an event either hasn't opened a *sql.Tx yet (for "before" events — the
// payload is an in-memory preview) or has already committed one (for "after"/"created"/
// "updated"/"deleted" events) before dispatching. This app runs SQLite through a single
// shared connection (db.SetMaxOpenConns(1)): if a callback issued its own DB call while
// the emitting operation's transaction still held that one connection, it would deadlock
// forever — the only connection is checked out by the very goroutine now blocked waiting
// for it. Guaranteeing no transaction is open during dispatch means a callback is always
// free to issue its own queries or transactions via the plugin's DB API.
//
// Most events ignore the value a callback returns. The "before" events, however, let a
// callback CANCEL the pending operation by returning a non-nil error — the operation
// checks the first non-nil error and aborts, propagating it to the caller. These
// cancellable hooks are:
//   - OnVoucherBatchEvent: EventVoucherBatchBeforeCreate, EventVoucherBatchBeforeDelete.
//   - OnVoucherEvent: EventVoucherBeforeCreate (cancels the batch before any DB writes), EventVoucherBeforeActivate.
//   - OnClientEvent: EventClientBeforeConnect, EventClientBeforeCreate,
//     EventClientBeforeUpdate, EventClientBeforeDisconnect.
//   - OnClientBeforeMerge: cancels a device merge (EventClientBeforeMerge).
//   - OnSessionEvent: EventSessionBeforeCreate, EventSessionBeforeConsume, EventSessionBeforeDelete.
//   - OnSessionBatchEvent: EventSessionBatchBeforeDelete, EventSessionBatchBeforeCreate.
//   - OnClientBatchEvent: EventClientBatchBeforeCreate.
//   - OnPurchaseEvent: EventPurchaseBeforeSuccess, EventPurchaseBeforeCancel (EventPurchaseBeforeFail
//     is notify-only — a failure cannot be vetoed).
//
// Example:
//
//	func Init(api sdkapi.IPluginApi) error {
//	    api.Events().OnSessionEvent(sdkapi.EventSessionCreated, func(ctx context.Context, data sdkapi.SessionEventData) error {
//	        // react to new session
//	        return nil
//	    })
//	    return nil
//	}
type IEventsApi interface {

	// OnSessionEvent registers a callback that fires whenever the given session
	// event occurs. The callback runs synchronously in the emitter's goroutine; its
	// returned error is ignored by the emitter. Spawn a goroutine if it must not block.
	//
	// The "before" events are cancellable: for EventSessionBeforeCreate,
	// EventSessionBeforeConsume, and EventSessionBeforeDelete, a callback that returns a
	// non-nil error aborts the pending operation (create/consume/delete) and the error
	// propagates to the caller. EventSessionBeforeCreate delivers an in-memory preview
	// session (ID == 0).
	//
	// Available events: EventSessionBeforeCreate, EventSessionCreated,
	// EventSessionConnected, EventSessionDisconnected, EventSessionBeforeConsume,
	// EventSessionConsumed, EventSessionChanged, EventSessionBeforeDelete,
	// EventSessionDeleted.
	OnSessionEvent(event SessionEvent, callback func(ctx context.Context, data SessionEventData) error)

	// OnSessionBatchEvent registers a callback that fires whenever a batch of
	// sessions is updated or about to be deleted at once. The callback runs synchronously
	// in the emitter's goroutine; spawn a goroutine if it must not block.
	//
	// EventSessionBatchBeforeDelete is cancellable: returning a non-nil error aborts the
	// whole batch deletion (from DeleteSessions) before any session is removed.
	// EventSessionBatchBeforeCreate is likewise cancellable: returning a non-nil error
	// aborts the whole batch creation (from CreateSessions) before any row is inserted.
	// For EventSessionBatchBeforeCreate the sessions are in-memory previews (ID == 0).
	//
	// Available events: EventSessionBatchUpdated, EventSessionBatchBeforeDelete,
	// EventSessionBatchBeforeCreate, EventSessionBatchCreated.
	OnSessionBatchEvent(event SessionEvent, callback func(ctx context.Context, sessions []IClientSession) error)

	// OnClientEvent registers a callback that fires whenever the given client
	// device event occurs. The callback runs synchronously in the emitter's goroutine,
	// in registration order; spawn a goroutine if it must not block. For most client
	// events the returned error is ignored.
	//
	// The "before" client events are cancellable: for EventClientBeforeConnect,
	// EventClientBeforeCreate, EventClientBeforeUpdate, and EventClientBeforeDisconnect,
	// a callback that returns a non-nil error aborts the pending operation and the error
	// propagates back to the caller. They fire before any side effects, so cancelling
	// requires no rollback. (Merges use OnClientBeforeMerge, not this method.)
	//
	// Available events: EventClientBeforeCreate, EventClientCreated,
	// EventClientRegistered, EventClientBeforeUpdate, EventClientUpdated,
	// EventClientConnected, EventClientBeforeDisconnect, EventClientDisconnected,
	// EventClientActive, EventClientBeforeConnect.
	OnClientEvent(event ClientEvent, callback func(ctx context.Context, clnt IClientDevice) error)

	// OnClientBatchEvent registers a callback that fires whenever a batch of client
	// devices is registered at once, from BatchRegisterClient(). The callback runs
	// synchronously in the emitter's goroutine; spawn a goroutine if it must not block.
	//
	// EventClientBatchBeforeCreate is cancellable: returning a non-nil error aborts
	// the whole batch registration before any row is inserted. The devices passed to
	// that event are in-memory previews (ID == 0) — the same objects the caller built
	// via NewClientDevice. EventClientBatchCreated fires once after the whole batch is
	// successfully committed; the per-device EventClientCreated and
	// EventClientRegistered (via OnClientEvent) also fire for each device in the batch.
	//
	// Available events: EventClientBatchBeforeCreate, EventClientBatchCreated.
	OnClientBatchEvent(event ClientBatchEvent, callback func(ctx context.Context, clients []IClientDevice) error)

	// OnClientBeforeMerge registers a callback that fires BEFORE two device records are
	// merged, while both still exist. The callback runs synchronously in the emitter's
	// goroutine. Returning a non-nil error CANCELS the merge before any data is
	// transferred or deleted (from an explicit MergeClientDevices call the error
	// propagates to the caller; from an implicit MAC-collision merge the merge is simply
	// skipped). Use EventClientMergeData.Target for the survivor and
	// EventClientMergeData.Source for the device about to be deleted — both are non-nil here.
	OnClientBeforeMerge(callback func(ctx context.Context, data EventClientMergeData) error)

	// OnClientMerge registers a callback that fires after two device records have
	// been successfully merged. The callback runs synchronously in the emitter's
	// goroutine; spawn a goroutine if it must not block.
	//
	// This event fires from multiple sources:
	//   - Real-time: MAC-collision detected during device registration
	//   - Scheduled: background duplicate-device merge job
	//   - Plugins calling api.SessionsMgr().MergeClientDevices()
	//
	// When the callback is invoked, the source device has already been deleted.
	// Use EventClientMergeData.Target to access the surviving device and
	// EventClientMergeData.SourceDeviceID for the ID of the deleted device.
	OnClientMerge(callback func(ctx context.Context, data EventClientMergeData) error)

	// OnPurchaseEvent registers a callback that fires whenever the given purchase
	// event occurs. The callback runs synchronously in the emitter's goroutine;
	// spawn a goroutine if it must not block.
	//
	// EventPurchaseBeforeRequest and EventPurchaseBeforeCancel are cancellable: a callback
	// that returns a non-nil error aborts the request/cancel before any side effects and
	// the error propagates to the caller.
	//
	// Available events: EventPurchaseBeforeRequest, EventPurchaseSuccess,
	// EventPurchaseFailed, EventPurchaseBeforeCancel, EventPurchaseCancelled.
	OnPurchaseEvent(event PurchaseEvent, callback func(ctx context.Context, data PurchaseEventData) error)

	// OnVoucherEvent registers a callback that fires whenever the given
	// single-voucher event occurs. The callback runs synchronously in the emitter's
	// goroutine; spawn a goroutine if it must not block.
	//
	// For EventVoucherBeforeCreate the voucher is an in-memory preview (ID == 0,
	// Code and UUID are already set). It fires once per voucher BEFORE the
	// creation transaction opens, so returning an error cancels the whole batch
	// before any row is inserted — no rollback is needed. EventVoucherBeforeActivate
	// is also cancellable: returning an error aborts the activation before any
	// session is created.
	//
	// Available events: EventVoucherBeforeCreate, EventVoucherBeforeActivate,
	// EventVoucherActivated, EventVoucherUpdated, EventVoucherDeleted.
	OnVoucherEvent(event VoucherEvent, callback func(ctx context.Context, v IVoucher) error)

	// OnVoucherBatchEvent registers a callback that fires whenever the given
	// voucher-batch event occurs. The callback runs synchronously in the emitter's
	// goroutine; spawn a goroutine if it must not block.
	//
	// For EventVoucherBatchBeforeCreate the batch is an in-memory preview (ID == 0,
	// Vouchers() returns nil). Returning an error cancels batch creation before
	// any DB writes. EventVoucherBatchBeforeDelete is likewise cancellable: returning an
	// error aborts the batch deletion before any row is removed.
	//
	// Available events: EventVoucherBatchBeforeCreate, EventVoucherBatchCreated,
	// EventVoucherBatchBeforeDelete, EventVoucherBatchDeleted.
	OnVoucherBatchEvent(event VoucherBatchEvent, callback func(ctx context.Context, batch IVoucherBatch) error)

	// OnInternetEvent registers a callback that fires whenever the device's
	// internet connectivity changes, as observed by the core's online monitor. The
	// callback runs synchronously in the monitor's goroutine, in registration
	// order; its returned error is logged but does not stop other callbacks. A
	// callback that does slow work (e.g. downloads, package installs) MUST spawn
	// its own goroutine so it does not stall the monitor's polling loop.
	//
	// Available events: EventInternetUp, EventInternetDown.
	OnInternetEvent(event InternetEvent, callback func(ctx context.Context) error)

	// OnBootEvent registers a callback that fires once the machine's boot sequence has
	// fully completed. The callback runs synchronously in the boot goroutine, in
	// registration order; its returned error is logged but does not stop other
	// callbacks. Spawn a goroutine if it must do slow work, so it does not stall
	// the rest of boot finalization.
	//
	// Available events: EventBoot.
	OnBootEvent(event BootEvent, callback func(ctx context.Context) error)
}
