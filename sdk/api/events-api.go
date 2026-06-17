/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "context"

// SessionEvent represents the type of a session event.
type SessionEvent string

// ClientEvent represents the type of a client device event.
type ClientEvent string

// PortalEvent represents the type of a portal event.
type PortalEvent string

// VoucherEvent represents the type of a voucher lifecycle event.
type VoucherEvent string

// PurchaseEvent represents the type of a purchase event.
type PurchaseEvent string

// PaymentEvent represents the type of a payment-related UI event.
type PaymentEvent string

// Session events.
const (
	EventSessionCreated      SessionEvent = "session:created"
	EventSessionConnected    SessionEvent = "session:connected"
	EventSessionDisconnected SessionEvent = "session:disconnected"
	EventSessionConsumed     SessionEvent = "session:expired"
	EventSessionChanged      SessionEvent = "session:changed"
	EventSessionDeleted      SessionEvent = "session:deleted"
	EventSessionBatchUpdated SessionEvent = "session:batch-updated"
)

// Client events.
const (
	EventClientCreated      ClientEvent = "client:created"
	EventClientRegistered   ClientEvent = "client:registered"
	EventClientUpdated      ClientEvent = "client:updated"
	EventClientConnected    ClientEvent = "client:connected"
	EventClientDisconnected ClientEvent = "client:disconnected"

	// EventClientActive is emitted when a known device shows network activity at
	// layer 3 — independently of whether it has a running session. The primary
	// source is the RFC 8908 captive portal API (advertised via DHCP option 114):
	// when a client's OS probes it, the device is provably on the network.
	// Subscribers use this as a "client connected" signal to drive auto-resume of
	// previously auto-paused sessions, mirroring a WiFi (re)association.
	EventClientActive ClientEvent = "client:active"

	// EventClientBeforeConnect is emitted SYNCHRONOUSLY before a client device is
	// connected to the internet, from the session manager's Connect() flow. Unlike
	// every other client event (which is async fire-and-forget), the callbacks for
	// this event run sequentially in the caller's goroutine: if any callback returns
	// an error, the connection is aborted and that error is propagated back to the
	// caller of Connect(). Use this as a veto hook (e.g. quota/credit checks, policy
	// enforcement). It fires before any side effects (firewall rules, session start),
	// so a veto requires no rollback. Callbacks must be fast and must not block
	// indefinitely.
	EventClientBeforeConnect ClientEvent = "client:before_connect"

	// EventClientMerge is emitted after two device records are successfully merged.
	// The source device (identified by Source) is deleted; the target device
	// (available as Target) is the one that was kept and received all transferred data.
	EventClientMerge ClientEvent = "client:merged"
)

// Voucher events.
const (
	// EventVoucherGenerated is emitted after a batch of vouchers is created.
	EventVoucherGenerated VoucherEvent = "voucher:generated"

	// EventVoucherActivated is emitted when a voucher is used to start a session.
	EventVoucherActivated VoucherEvent = "voucher:activated"

	// EventVoucherUpdated is emitted when a voucher's validity is updated.
	EventVoucherUpdated VoucherEvent = "voucher:updated"

	// EventVoucherDeleted is emitted when a voucher is deleted.
	EventVoucherDeleted VoucherEvent = "voucher:deleted"

	// EventVoucherBatchDeleted is emitted when a voucher batch is deleted.
	EventVoucherBatchDeleted VoucherEvent = "voucher:batch_deleted"
)

// Note: the pre-create veto hook is NOT a VoucherEvent constant. Because it must
// run synchronously and be able to block creation, it is registered via the
// dedicated IEventsApi.OnVoucherBeforeCreate method rather than the async,
// event-keyed OnVoucherEvent path.

// Purchase events.
const (
	// EventPurchaseSuccess is emitted when a purchase is successfully confirmed.
	EventPurchaseSuccess PurchaseEvent = "purchase:success"

	// EventPurchaseFailed is emitted when a purchase confirmation or execution fails.
	EventPurchaseFailed PurchaseEvent = "purchase:failed"

	// EventPurchaseCancelled is emitted when a purchase is cancelled by the user.
	EventPurchaseCancelled PurchaseEvent = "purchase:cancelled"
)

// Payment events.
const (
	// EventPaymentOptionsChanged is emitted when the list of available payment
	// options changes. This occurs when payment providers become available or
	// unavailable (e.g., devices going online/offline).
	EventPaymentOptionsChanged PaymentEvent = "payment:options:changed"
)

// IEventsApi is the unified event subscription API for plugins.
//
// All registration methods are safe to call concurrently. Callbacks are dispatched
// asynchronously (each in its own goroutine).
//
// Prefer this API over the individual On* methods on ISessionsMgrApi, IVouchersApi,
// and IPaymentsApi, which are deprecated.
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
	// event occurs. The callback runs asynchronously; errors are logged but not
	// propagated to the caller that emitted the event.
	//
	// Available events: EventSessionCreated, EventSessionConnected,
	// EventSessionDisconnected, EventSessionConsumed, EventSessionChanged,
	// EventSessionDeleted.
	OnSessionEvent(event SessionEvent, callback func(ctx context.Context, data SessionEventData) error)

	// OnSessionBatchEvent registers a callback that fires whenever a batch of
	// sessions is updated at once. The callback runs asynchronously.
	//
	// Available events: EventSessionBatchUpdated.
	OnSessionBatchEvent(event SessionEvent, callback func(ctx context.Context, sessions []IClientSession) error)

	// OnClientEvent registers a callback that fires whenever the given client
	// device event occurs. For most events the callback runs asynchronously and a
	// returned error is logged but ignored.
	//
	// EXCEPTION — EventClientBeforeConnect: callbacks for this event run
	// SYNCHRONOUSLY in the Connect() caller's goroutine, in registration order. If a
	// callback returns an error, the connection is aborted and the error propagates
	// back to the caller of Connect(). Use it to veto a connection.
	//
	// Available events: EventClientCreated, EventClientRegistered,
	// EventClientUpdated, EventClientConnected, EventClientDisconnected,
	// EventClientActive, EventClientBeforeConnect.
	OnClientEvent(event ClientEvent, callback func(ctx context.Context, clnt IClientDevice) error)

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
	OnClientMerge(callback func(ctx context.Context, data EventClientMergeData) error)

	// OnPurchaseEvent registers a callback that fires whenever the given purchase
	// event occurs. The callback runs asynchronously.
	//
	// Available events: EventPurchaseSuccess, EventPurchaseFailed,
	// EventPurchaseCancelled.
	OnPurchaseEvent(event PurchaseEvent, callback func(ctx context.Context, data PurchaseEventData) error)

	// OnVoucherEvent registers a callback that fires whenever the given
	// single-voucher event occurs. The callback runs asynchronously.
	//
	// Available events: EventVoucherActivated, EventVoucherUpdated,
	// EventVoucherDeleted.
	OnVoucherEvent(event VoucherEvent, callback func(ctx context.Context, v IVoucher) error)

	// OnVoucherBatchEvent registers a callback that fires whenever the given
	// voucher-batch event occurs. The callback runs asynchronously.
	//
	// Available events: EventVoucherGenerated, EventVoucherBatchDeleted.
	OnVoucherBatchEvent(event VoucherEvent, callback func(ctx context.Context, batch IVoucherBatch) error)

	// OnVoucherBeforeCreate registers a SYNCHRONOUS hook that runs before a batch
	// of vouchers is created. Unlike the async On*Event methods, hooks run
	// sequentially in the CreateVouchers caller's goroutine, in registration order.
	// Each hook receives a pointer to the (already defaulted) creation params and
	// may modify them — e.g. clamp Count or override defaults. If any hook returns
	// a non-nil error, creation is aborted and the error is returned to the caller
	// of CreateVouchers. Use this for quota/credit checks or policy enforcement.
	// BatchUUID and bandwidth defaults are guaranteed to be populated before hooks run.
	OnVoucherBeforeCreate(callback func(ctx context.Context, params *CreateVouchersParams) error)
}
