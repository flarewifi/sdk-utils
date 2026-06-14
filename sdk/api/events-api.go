/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "context"

// IEventsApi is the unified event subscription API for plugins.
//
// All registration methods are safe to call concurrently. Callbacks are dispatched
// asynchronously (each in its own goroutine) except for OnBeforeCreate, which is
// synchronous and can block or abort voucher creation by returning a non-nil error.
//
// Prefer this API over the individual On* methods on ISessionsMgrApi, IVouchersApi,
// and IPaymentsApi, which are deprecated.
//
// Example:
//
//	func Init(api sdkapi.IPluginApi) error {
//	    api.Events().OnSessionEvent(sdkapi.EventSessionCreated, func(data sdkapi.SessionEventData) error {
//	        // react to new session
//	        return nil
//	    })
//	    return nil
//	}
type IEventsApi interface {

	// OnSessionEvent registers a callback that fires whenever the given session
	// event occurs. The callback runs asynchronously; errors are logged but not
	// propagated to the caller that emitted the event.
	//
	// Available events: EventSessionCreated, EventSessionConnected,
	// EventSessionDisconnected, EventSessionConsumed, EventSessionChanged,
	// EventSessionDeleted.
	OnSessionEvent(event SessionEvent, callback func(data SessionEventData) error)

	// OnSessionBatchEvent registers a callback that fires whenever a batch of
	// sessions is updated at once. The callback runs asynchronously.
	//
	// Available events: EventSessionBatchUpdated.
	OnSessionBatchEvent(event SessionEvent, callback func(sessions []IClientSession) error)

	// OnClientEvent registers a callback that fires whenever the given client
	// device event occurs. The callback runs asynchronously.
	//
	// Available events: EventClientCreated, EventClientRegistered,
	// EventClientUpdated, EventClientConnected, EventClientDisconnected,
	// EventClientActive.
	OnClientEvent(event ClientEvent, callback func(clnt IClientDevice) error)

	// OnClientMerge registers a callback that fires after two device records have
	// been successfully merged. The callback runs asynchronously.
	//
	// This event fires from multiple sources:
	//   - Real-time: MAC-collision detected during device registration
	//   - Scheduled: background duplicate-device merge job
	//   - Plugins calling api.SessionsMgr().MergeClientDevices()
	//
	// When the callback is invoked, the source device has already been deleted.
	// Use EventClientMergeData.Target to access the surviving device and
	// EventClientMergeData.SourceDeviceID for the ID of the deleted device.
	OnClientMerge(callback func(data EventClientMergeData) error)

	// OnPurchaseEvent registers a callback that fires whenever the given purchase
	// event occurs. The callback runs asynchronously.
	//
	// Available events: EventPurchaseSuccess, EventPurchaseFailed,
	// EventPurchaseCancelled.
	OnPurchaseEvent(event PurchaseEvent, callback func(data PurchaseEventData) error)

	// OnVoucherEvent registers a callback that fires whenever the given
	// single-voucher event occurs. The callback runs asynchronously.
	//
	// Available events: EventVoucherActivated, EventVoucherUpdated,
	// EventVoucherDeleted.
	OnVoucherEvent(event VoucherEvent, callback func(IVoucher) error)

	// OnVoucherBatchEvent registers a callback that fires whenever the given
	// voucher-batch event occurs. The callback runs asynchronously.
	//
	// Available events: EventVoucherGenerated, EventVoucherBatchDeleted.
	OnVoucherBatchEvent(event VoucherEvent, callback func(IVoucherBatch) error)

	// OnBeforeCreate registers a hook that is called synchronously before vouchers
	// are created. Hooks run in registration order; the first hook that returns a
	// non-nil error aborts the chain and prevents voucher creation.
	//
	// The hook receives a pointer to the creation params and may modify them
	// (e.g. to enforce quota limits or override defaults).
	OnBeforeCreate(callback func(ctx context.Context, params *CreateVouchersParams) error)
}
