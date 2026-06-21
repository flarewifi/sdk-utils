/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

// SessionEvent represents the type of a session event.
type SessionEvent string

// ClientEvent represents the type of a client device event.
type ClientEvent string

// PortalEvent represents the type of a portal event.
type PortalEvent string

// VoucherEvent represents the type of a single-voucher lifecycle event.
type VoucherEvent string

// VoucherBatchEvent represents the type of a voucher-batch lifecycle event.
type VoucherBatchEvent string

// PurchaseEvent represents the type of a purchase event.
type PurchaseEvent string

// PaymentEvent represents the type of a payment-related UI event.
type PaymentEvent string

// InternetEvent represents a change in the device's internet connectivity, as
// observed by the core's online monitor.
type InternetEvent string

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

	// EventClientBeforeConnect is emitted before a client device is connected to the
	// internet, from the session manager's Connect() flow. Like all events its
	// callbacks run synchronously; what makes it special is that Connect() checks the
	// returned error: if any callback returns an error, the connection is cancelled
	// and that error is propagated back to the caller of Connect(). Use this for
	// quota/credit checks or policy enforcement. It fires before any side effects
	// (firewall rules, session start), so cancelling requires no rollback. Callbacks
	// must be fast and must not block indefinitely.
	EventClientBeforeConnect ClientEvent = "client:before_connect"

	// EventClientMerge is emitted after two device records are successfully merged.
	// The source device (identified by Source) is deleted; the target device
	// (available as Target) is the one that was kept and received all transferred data.
	EventClientMerge ClientEvent = "client:merged"
)

// Single-voucher events (used with OnVoucherEvent).
const (
	// EventVoucherBeforeCreate fires synchronously for each voucher before its
	// INSERT, inside the batch transaction. The voucher is an in-memory preview
	// (ID == 0). Returning an error rolls back the entire batch transaction.
	EventVoucherBeforeCreate VoucherEvent = "voucher:before_create"

	// EventVoucherActivated is emitted when a voucher is used to start a session.
	EventVoucherActivated VoucherEvent = "voucher:activated"

	// EventVoucherUpdated is emitted when a voucher's validity is updated.
	EventVoucherUpdated VoucherEvent = "voucher:updated"

	// EventVoucherDeleted is emitted when a voucher is deleted.
	EventVoucherDeleted VoucherEvent = "voucher:deleted"
)

// Voucher-batch events (used with OnVoucherBatchEvent).
const (
	// EventVoucherBatchBeforeCreate fires synchronously once before any DB writes
	// for a batch. The batch is an in-memory preview (ID == 0, Vouchers() is nil).
	// Returning an error cancels creation with no rollback needed.
	EventVoucherBatchBeforeCreate VoucherBatchEvent = "voucher:before_create"

	// EventVoucherGenerated is emitted after a batch of vouchers is successfully created.
	EventVoucherGenerated VoucherBatchEvent = "voucher:generated"

	// EventVoucherBatchDeleted is emitted when a voucher batch is deleted.
	EventVoucherBatchDeleted VoucherBatchEvent = "voucher:batch_deleted"
)

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

// Internet connectivity events.
const (
	// EventInternetUp is emitted when the core's online monitor observes that the
	// device has gained internet access — either at boot once connectivity first
	// arrives, or after a previous outage is restored. The core uses this signal
	// to run install work that needs the network (a plugin's system_packages and
	// its preinstall/postinstall scripts), so a device flashed offline still gets
	// fully provisioned the moment it reaches the internet. Because the callback
	// may run long (opkg/pip), spawn a goroutine if it must not block the monitor.
	EventInternetUp InternetEvent = "internet:up"

	// EventInternetDown is emitted when the online monitor observes that internet
	// access has been lost. Use it to pause network-dependent work or surface an
	// offline state in the UI.
	EventInternetDown InternetEvent = "internet:down"
)

// SessionEventData represents the data associated with a session event.
type SessionEventData struct {
	Session       IClientSession
	ChangedFields SessionChangedFields // Which fields changed (only set for EventSessionChanged)
}

// EventClientMergeData carries the context of a device-merge event.
type EventClientMergeData struct {
	// Target is the device that was kept after the merge. All sessions, purchases,
	// fingerprints, and wallet balance from the source device have been transferred to it.
	Target IClientDevice

	// SourceDeviceID is the database ID of the device that was deleted during the merge.
	// The source device no longer exists in the database when callbacks are invoked.
	SourceDeviceID int64

	// SourceDeviceUUID is the local UUID of the device that was deleted during the merge.
	// Captured before deletion so plugins can notify external systems (e.g. cloud sync).
	SourceDeviceUUID string
}

// PurchaseEventData represents the data associated with a purchase event.
type PurchaseEventData struct {
	// Purchase is the purchase request that triggered the event.
	Purchase IPurchaseRequest

	// Device is the client device associated with the purchase.
	Device IClientDevice

	// Reason provides context for failure or cancellation events.
	// Empty for success events.
	Reason string
}
