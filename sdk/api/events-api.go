/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "context"

// IEventsApi is the unified event subscription API for plugins.
//
// All registration methods are safe to call concurrently.
//
// DISPATCH IS SYNCHRONOUS. When an event fires, its registered callbacks run
// sequentially in the emitter's goroutine, in registration order. A callback that
// must not block the emitting operation — or that runs long — should spawn its own
// goroutine: deciding sync vs async is the handler's responsibility, not the core's.
//
// Most events ignore the value a callback returns. A few let a callback cancel the
// operation by returning an error — the operation checks the first non-nil error:
//   - OnVoucherBatchEvent for EventVoucherBeforeCreate — an error cancels voucher creation.
//   - OnVoucherEvent for EventVoucherBeforeCreate — an error cancels the current voucher (rolls back the batch transaction).
//   - OnClientEvent for EventClientBeforeConnect — an error cancels the connection.
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
	// Available events: EventSessionCreated, EventSessionConnected,
	// EventSessionDisconnected, EventSessionConsumed, EventSessionChanged,
	// EventSessionDeleted.
	OnSessionEvent(event SessionEvent, callback func(ctx context.Context, data SessionEventData) error)

	// OnSessionBatchEvent registers a callback that fires whenever a batch of
	// sessions is updated at once. The callback runs synchronously in the emitter's
	// goroutine; spawn a goroutine if it must not block.
	//
	// Available events: EventSessionBatchUpdated.
	OnSessionBatchEvent(event SessionEvent, callback func(ctx context.Context, sessions []IClientSession) error)

	// OnClientEvent registers a callback that fires whenever the given client
	// device event occurs. The callback runs synchronously in the emitter's goroutine,
	// in registration order; spawn a goroutine if it must not block. For most client
	// events the returned error is ignored.
	//
	// EventClientBeforeConnect is the exception: Connect() honors the returned error.
	// If a callback returns an error, the connection is aborted and the error
	// propagates back to the caller of Connect(). Use it to stop a connection.
	//
	// Available events: EventClientCreated, EventClientRegistered,
	// EventClientUpdated, EventClientConnected, EventClientDisconnected,
	// EventClientActive, EventClientBeforeConnect.
	OnClientEvent(event ClientEvent, callback func(ctx context.Context, clnt IClientDevice) error)

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
	// Available events: EventPurchaseSuccess, EventPurchaseFailed,
	// EventPurchaseCancelled.
	OnPurchaseEvent(event PurchaseEvent, callback func(ctx context.Context, data PurchaseEventData) error)

	// OnVoucherEvent registers a callback that fires whenever the given
	// single-voucher event occurs. The callback runs synchronously in the emitter's
	// goroutine; spawn a goroutine if it must not block.
	//
	// For EventVoucherBeforeCreate the voucher is an in-memory preview (ID == 0,
	// Code and UUID are already set). Returning an error rolls back the whole
	// batch transaction.
	//
	// Available events: EventVoucherBeforeCreate, EventVoucherActivated,
	// EventVoucherUpdated, EventVoucherDeleted.
	OnVoucherEvent(event VoucherEvent, callback func(ctx context.Context, v IVoucher) error)

	// OnVoucherBatchEvent registers a callback that fires whenever the given
	// voucher-batch event occurs. The callback runs synchronously in the emitter's
	// goroutine; spawn a goroutine if it must not block.
	//
	// For EventVoucherBatchBeforeCreate the batch is an in-memory preview (ID == 0,
	// Vouchers() returns nil). Returning an error cancels batch creation before
	// any DB writes.
	//
	// Available events: EventVoucherBatchBeforeCreate, EventVoucherGenerated,
	// EventVoucherBatchDeleted.
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

	// OnBoot registers a callback that fires once the machine's boot sequence has
	// fully completed. The callback runs synchronously in the boot goroutine, in
	// registration order; its returned error is logged but does not stop other
	// callbacks. Spawn a goroutine if it must do slow work, so it does not stall
	// the rest of boot finalization.
	//
	// Available events: EventBoot.
	OnBoot(event BootEvent, callback func(ctx context.Context) error)
}
