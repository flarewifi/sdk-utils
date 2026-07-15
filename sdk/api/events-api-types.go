/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "time"

// SessionEvent represents the type of a session event.
type SessionEvent string

// ClientEvent represents the type of a client device event.
type ClientEvent string

// ClientBatchEvent represents the type of a client-device-batch lifecycle event.
type ClientBatchEvent string

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

// BootEvent represents a milestone in the machine's boot sequence.
type BootEvent string

// DhcpEvent represents a DHCPv4 lease lifecycle event, reported by dnsmasq's
// dhcp-script hook (see https://openwrt.org/docs/guide-user/base-system/dhcp).
// IPv6 leases are handled by odhcpd on this machine, not dnsmasq, so they are
// not covered here.
type DhcpEvent string

// Session events.
const (
	EventSessionCreated      SessionEvent = "session:created"
	EventSessionConnected    SessionEvent = "session:connected"
	EventSessionDisconnected SessionEvent = "session:disconnected"
	EventSessionConsumed     SessionEvent = "session:expired"
	EventSessionChanged      SessionEvent = "session:changed"
	EventSessionDeleted      SessionEvent = "session:deleted"
	EventSessionBatchUpdated SessionEvent = "session:batch-updated"

	// EventSessionBeforeCreate is emitted before a session row is inserted, from
	// CreateSession(). The session is an in-memory preview (ID == 0). Returning an error
	// from any callback cancels creation before the INSERT, so no rollback is needed.
	EventSessionBeforeCreate SessionEvent = "session:before_create"

	// EventSessionBeforeConsume is emitted before a running session is finalized as
	// consumed (time/data exhausted), from RunningSession.StopWithReason. It fires before
	// the session is persisted and the terminal EventSessionConsumed is emitted, while the
	// session is still running and its timer intact. Returning an error from any callback
	// vetoes consumption: the session is left running so a plugin can, for example, top it
	// up. A vetoing plugin is responsible for extending time/data and re-arming enforcement.
	EventSessionBeforeConsume SessionEvent = "session:before_consume"

	// EventSessionBeforeDelete is emitted before a single session is deleted, from
	// DeleteSession(). Returning an error from any callback cancels the deletion before
	// the device is disconnected or the row removed.
	EventSessionBeforeDelete SessionEvent = "session:before_delete"

	// EventSessionBatchBeforeDelete is emitted once before a batch of sessions is deleted,
	// from DeleteSessions(). Subscribe via OnSessionBatchEvent. Returning an error from any
	// callback cancels the whole batch deletion before any session is removed.
	EventSessionBatchBeforeDelete SessionEvent = "session:batch_before_delete"

	// EventSessionBatchBeforeCreate is emitted once before any DB writes for a batch
	// of sessions, from CreateSessions(). The sessions are in-memory previews (ID ==
	// 0). Returning an error from any callback cancels the whole batch before any
	// row is inserted, so no rollback is needed. The single-session
	// EventSessionBeforeCreate is not fired per item.
	EventSessionBatchBeforeCreate SessionEvent = "session:batch_before_create"

	// EventSessionBatchCreated is emitted after a batch of sessions is successfully
	// created, from CreateSessions(). The per-session EventSessionCreated is also
	// fired for each session in the batch.
	EventSessionBatchCreated SessionEvent = "session:batch_created"
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

	// EventClientBeforeCreate is emitted before a brand-new device record is inserted,
	// from the client register. The device is an in-memory preview (ID == 0). Returning
	// an error from any callback cancels registration before the INSERT, so no rollback
	// is needed. Use it for admission control (e.g. block-list, capacity limits).
	EventClientBeforeCreate ClientEvent = "client:before_create"

	// EventClientBeforeUpdate is emitted before a device's network details are written,
	// from UpdateDevice(). The device still carries its pre-update values. Returning an
	// error from any callback cancels the update before any DB write or reconnect.
	EventClientBeforeUpdate ClientEvent = "client:before_update"

	// EventClientBeforeDisconnect is emitted before a device is disconnected from the
	// internet, from SessionsMgr.Disconnect(). It fires before any teardown (firewall,
	// TC, session stop). Returning an error from any callback cancels the disconnect,
	// so no rollback is needed, and the error propagates back to the caller.
	EventClientBeforeDisconnect ClientEvent = "client:before_disconnect"

	// EventClientBeforeMerge is emitted before two device records are merged, while both
	// still exist. Subscribe via OnClientBeforeMerge (merges do not use the ClientEvent
	// map). The event carries both devices via EventClientMergeData.Target (the survivor)
	// and EventClientMergeData.Source (the device about to be deleted). Returning an error
	// cancels the merge before any data is transferred or deleted. Compare with the
	// post-merge EventClientMerge, where Source is already gone.
	EventClientBeforeMerge ClientEvent = "client:before_merge"

	// EventClientMerge is emitted after two device records are successfully merged.
	// The source device (identified by SourceDeviceID/SourceDeviceUUID) is deleted; the
	// target device (available as Target) is the one that was kept and received all
	// transferred data. Subscribe via OnClientMerge.
	EventClientMerge ClientEvent = "client:merged"
)

// Client-batch events (used with OnClientBatchEvent).
const (
	// EventClientBatchBeforeCreate fires synchronously once before any DB writes for
	// a batch of client devices, from BatchRegisterClient(). The devices are the
	// in-memory previews passed in (ID == 0). Returning an error from any callback
	// cancels the whole batch before any row is inserted, so no rollback is needed.
	EventClientBatchBeforeCreate ClientBatchEvent = "client:batch_before_create"

	// EventClientBatchCreated is emitted after a batch of client devices is
	// successfully registered, from BatchRegisterClient(). The per-device
	// EventClientCreated and EventClientRegistered are also fired for each device
	// in the batch.
	EventClientBatchCreated ClientBatchEvent = "client:batch_created"
)

// Single-voucher events (used with OnVoucherEvent).
const (
	// EventVoucherBeforeCreate fires synchronously for each voucher, once for
	// every voucher in the batch, BEFORE the creation transaction opens (so a
	// subscriber's own DB call is never made while this call holds the write
	// transaction — this app runs SQLite through a single shared connection,
	// see db.SetMaxOpenConns(1), so that would block forever). The voucher is
	// an in-memory preview (ID == 0). Returning an error cancels the whole
	// batch before any row is inserted, so no rollback is needed.
	EventVoucherBeforeCreate VoucherEvent = "voucher:before_create"

	// EventVoucherBeforeActivate is emitted before a voucher is activated (used to start a
	// session), from Activate(). It fires before the session is created and the voucher is
	// marked activated. Returning an error from any callback cancels activation before any
	// side effects, so no rollback is needed.
	EventVoucherBeforeActivate VoucherEvent = "voucher:before_activate"

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

	// EventVoucherBatchCreated is emitted after a batch of vouchers is successfully
	// created. (Renamed from EventVoucherGenerated to match the created/updated/deleted
	// naming used elsewhere.)
	EventVoucherBatchCreated VoucherBatchEvent = "voucher:batch_created"

	// EventVoucherBatchBeforeDelete is emitted before a voucher batch is deleted, from
	// DeleteBatch(). Returning an error from any callback cancels the deletion before any
	// row is removed.
	EventVoucherBatchBeforeDelete VoucherBatchEvent = "voucher:batch_before_delete"

	// EventVoucherBatchDeleted is emitted when a voucher batch is deleted.
	EventVoucherBatchDeleted VoucherBatchEvent = "voucher:batch_deleted"
)

// Purchase events.
const (
	// EventPurchaseBeforeRequest is emitted before a purchase request is created, from
	// CreatePayment(). It fires before the purchase record and payment are set up. Returning
	// an error from any callback cancels the request before any side effects, and the error
	// propagates to the caller. Use it for eligibility, quota, or block-list checks.
	EventPurchaseBeforeRequest PurchaseEvent = "purchase:before_request"

	// EventPurchaseBeforeCancel is emitted before a purchase is cancelled, from Cancel().
	// Returning an error from any callback cancels the cancellation before any side effects,
	// and the error propagates to the caller.
	EventPurchaseBeforeCancel PurchaseEvent = "purchase:before_cancelled"

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

// Boot events.
const (
	// EventBoot is emitted once, after the boot sequence has fully completed (the
	// captive portal is up, the network is initialized, and provisioning has run or
	// been deferred). The core uses it to defer the online monitor's connectivity
	// emissions until boot is done, so a still-initializing WAN during reboot cannot
	// surface a spurious "no internet" notification. Plugins can subscribe to run
	// work that should only start once the machine is fully booted.
	EventBoot BootEvent = "boot:complete"
)

// DHCPv4 lease events (used with OnDhcpEvent). These map 1:1 onto the three
// actions dnsmasq's dhcp-script hook invokes as its $1 argument.
const (
	// EventDhcpLeaseAdd fires when dnsmasq hands a brand-new lease to a client.
	// Data.Hostname is populated if the client supplied one.
	EventDhcpLeaseAdd DhcpEvent = "dhcp:lease_add"

	// EventDhcpLeaseOld fires for an existing lease: a client renewal/rebind, or a
	// replay of every current lease when dnsmasq itself starts or reloads. Data.Hostname
	// is populated only when a client actually resumed/renewed — not on a cold dnsmasq
	// restart replay, since dnsmasq does not persist hostnames in its lease database.
	EventDhcpLeaseOld DhcpEvent = "dhcp:lease_old"

	// EventDhcpLeaseDel fires when a lease is destroyed: released by the client,
	// expired, or removed administratively. Data.Hostname is never populated.
	EventDhcpLeaseDel DhcpEvent = "dhcp:lease_del"
)

// DhcpEventData carries the lease details dnsmasq passed to its dhcp-script hook
// for a single DHCPv4 event.
type DhcpEventData struct {
	// Mac is the client's hardware MAC address.
	Mac string

	// Ip is the leased IPv4 address.
	Ip string

	// Hostname is the hostname the client supplied, if any — see the per-event doc
	// comments above for when dnsmasq omits it.
	Hostname string

	// Interface is the name of the interface the DHCP request arrived on (e.g.
	// "br-lan"). Empty for an EventDhcpLeaseOld replay emitted when dnsmasq itself
	// restarts, since there is no live request to attribute it to.
	Interface string

	// Tags lists the dnsmasq config tags matched for this DHCP transaction (see
	// OpenWrt's dhcp "tag"/"tag:" option), space-separated as dnsmasq supplies them.
	// Empty if no tags matched.
	Tags string

	// LeaseExpires is the lease's expiry time, computed from dnsmasq's
	// DNSMASQ_TIME_REMAINING (seconds remaining, always set regardless of whether the
	// RTC-dependent DNSMASQ_LEASE_EXPIRES is available) at the moment the event was
	// observed. Zero for EventDhcpLeaseDel, where the lease has no remaining time.
	LeaseExpires time.Time
}

// SessionEventData represents the data associated with a session event.
type SessionEventData struct {
	Session       IClientSession
	ChangedFields SessionChangedFields // Which fields changed (only set for EventSessionChanged)
}

// EventClientMergeData carries the context of a device-merge event.
type EventClientMergeData struct {
	// Target is the surviving device. For EventClientMerge (after) it has already
	// received all sessions, purchases, and fingerprints from the source device; for
	// EventClientBeforeMerge (before) the transfer has not happened yet.
	Target IClientDevice

	// Source is the device that is about to be deleted. It is populated ONLY for the
	// pre-merge EventClientBeforeMerge event, where the device still exists. For the
	// post-merge EventClientMerge event it is nil (the row is already gone) — use
	// SourceDeviceID/SourceDeviceUUID there instead.
	Source IClientDevice

	// SourceDeviceID is the database ID of the device that was (or will be) deleted.
	SourceDeviceID int64

	// SourceDeviceUUID is the local UUID of the device that was (or will be) deleted.
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
