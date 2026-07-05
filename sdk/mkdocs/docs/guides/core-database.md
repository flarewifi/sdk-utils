# Core Database Tables

This page documents the **core system** database schema — the tables owned by the
Flarewifi core.

!!! note "Core vs plugin tables"
    Only the tables below are core-owned. Plugins create their **own** tables and
    reference core tables via foreign keys — they never alter core tables. See the
    [Plugin Development](../index.md) guides for plugin schemas.

!!! warning "Timestamps are UTC"
    **All** timestamps are stored in **UTC**.
    Convert to local time in Go for display with `sdkutil.UtcToLocalTime(t)`.

## Conventions

| Convention | Meaning |
|------------|---------|
| `id` | `INTEGER PRIMARY KEY` — local auto-increment row id (use `int64` in Go). |
| `uuid` | Stable, globally-unique string id. Survives cloud-sync where local `id`s differ. |
| `*_at` | `TIMESTAMP` in UTC. A `NULL` value usually means "not yet" (e.g. `activated_at IS NULL` = not activated). |
| `ON DELETE CASCADE` | Child rows are removed when the parent is deleted. |

---

## Table index

| Table | Purpose |
|-------|---------|
| [`devices`](#devices) | Client devices (phones/laptops) connecting through the machine. |
| [`device_macs`](#device_macs) | MAC-address history per device (one device may roam MACs). |
| [`device_current_macs`](#device_current_macs) | **View** — current MAC of each device. |
| [`device_fingerprints`](#device_fingerprints) | Browser/OS fingerprints used to identify a device. |
| [`device_logs`](#device_logs) | Per-device activity log entries. |
| [`sessions`](#sessions) | Internet access sessions (time/data quota & consumption). |
| [`vouchers`](#vouchers) | Redeemable access codes that generate a session. |
| [`voucher_batches`](#voucher_batches) | Groups of vouchers generated together. |
| [`purchases`](#purchases) | In-app purchase orders (plugin checkout). |
| [`payments`](#payments) | Payment records settling a purchase. |
| [`notifications`](#notifications) | Admin dashboard notifications. |

---

## Client identity

### `devices`

A **client device** (phone, laptop, etc.) that connects *through* the machine.
This is the central entity most other tables reference.

**Used by:** [IClientDevice](../api/client-device.md) · [ISessionsMgrApi](../api/sessions-mgr-api.md)

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Local device id. |
| `uuid` | VARCHAR(36) | Stable device id (unique when non-empty). |
| `ipv4_addr` | VARCHAR(15) | Current IPv4 address (`''` if none). |
| `ipv6_addr` | VARCHAR(45) | Current IPv6 address (`''` if none). |
| `hostname` | VARCHAR(64) | Reported client hostname. |
| `status` | INTEGER | Connection status — `1` Connected, `2` Disconnected (default), `3` Blocked. |
| `cookie_token` | VARCHAR | Token stored in the client's browser cookie to re-identify it. |
| `created_at` / `updated_at` | TIMESTAMP | Row lifecycle. |

!!! info "Dual-stack & MAC history"
    A device's MAC addresses live in [`device_macs`](#device_macs) (a device may use
    several over time), and its IP is split across `ipv4_addr` / `ipv6_addr`.
    Application-layer validation ensures at least one IP column is non-empty.

### `device_macs`

Full **MAC-address history** for a device. A single client may be seen under
several MACs over time (e.g. MAC randomization), so each MAC is tracked here as
its own row.

**Used by:** [ISessionsMgrApi.FindClientByMac](../api/sessions-mgr-api.md#findclientbymac) · [IClientDevice.MacAddr](../api/client-device.md#macaddr)

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Row id. |
| `device_id` | INTEGER → `devices.id` | Owning device (cascade delete). |
| `mac_address` | VARCHAR(17) | The MAC. Unique per `(device_id, mac_address)`. |
| `is_current` | BOOLEAN | `TRUE` for the device's active MAC. |
| `first_seen_at` / `last_seen_at` | TIMESTAMP | First/last time this MAC was observed. |

### `device_current_macs`

A **read-only view** (not a table) exposing only the current MAC of each device —
a convenience over `device_macs WHERE is_current = TRUE`.

**Used by:** [ISessionsMgrApi.FindClientByMac](../api/sessions-mgr-api.md#findclientbymac)

| Column | Description |
|--------|-------------|
| `device_id` | Device id. |
| `mac_address` | The current MAC. |
| `last_seen_at` | When it was last seen. |

### `device_fingerprints`

Browser/OS **fingerprints** used to recognize a returning device even when its
IP or MAC changes.

**Used by:** [ISessionsMgrApi](../api/sessions-mgr-api.md) (device matching & merge)

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Row id. |
| `device_id` | INTEGER → `devices.id` | Owning device (cascade delete). |
| `fingerprint_hash` | VARCHAR(64) | Hash identifying the fingerprint. |
| `user_agent` | TEXT | Raw User-Agent string. |
| `browser_name` | VARCHAR(50) | Parsed browser name. |
| `os_family` | VARCHAR(50) | Parsed OS family. |
| `screen_resolution` | VARCHAR(20) | Reported screen resolution. |
| `language` | VARCHAR(10) | Browser language. |
| `timezone` | VARCHAR | Browser-reported timezone. |
| `is_cna` | BOOLEAN | `TRUE` if seen via the OS Captive Network Assistant (the captive-portal mini-browser) rather than a full browser. |
| `created_at` / `last_seen_at` | TIMESTAMP | First/last time this fingerprint was seen. |

### `device_logs`

Per-device **activity log** entries surfaced in the admin device view.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Row id. |
| `device_id` | INTEGER → `devices.id` | Owning device (cascade delete). |
| `message` | TEXT | Human-readable log message. |
| `metadata` | TEXT | JSON blob of structured context (default `{}`). |
| `created_at` | DATETIME | When logged (UTC). |

---

## Sessions

### `sessions`

An **internet access session** — the quota (time and/or data) granted to a device
and how much has been consumed. Sessions are the unit hotspot plugins create when
a device pays, redeems a voucher, etc.

**Used by:** [IClientSession](../api/client-session.md) · [ISessionsMgrApi](../api/sessions-mgr-api.md)

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Local session id. |
| `uuid` | VARCHAR(255) | Stable session id (unique). Prefer this over `id` for cross-references — cloud-sync sessions may lack a local `id`. |
| `provider_pkg` | VARCHAR(255) | Plugin package that created the session. |
| `device_id` | INTEGER → `devices.id` | Owning device (cascade delete). |
| `session_type` | VARCHAR(20) | `time` or `data` — which quota governs the session. |
| `time_secs` | INT | Granted time quota, seconds. |
| `data_mbytes` | DECIMAL | Granted data quota, MB. |
| `consumption_secs` | INT | Time consumed so far, seconds. |
| `consumption_mb` | DECIMAL | Data consumed so far, MB. |
| `exp_days` | INT (nullable) | Days until the session expires (`NULL` = no expiry). |
| `down_mbits` / `up_mbits` | INT | Download/upload speed caps, Mbps (`0` = unlimited). |
| `use_global` | BOOLEAN | Whether the session draws from a global/shared pool. |
| `started_at` | TIMESTAMP (nullable) | When the session first started running. |
| `resumed_at` | TIMESTAMP (nullable) | When last resumed after a pause. |
| `created_at` / `updated_at` | TIMESTAMP | Row lifecycle. |

---

## Vouchers

### `vouchers`

A redeemable **access code**. When redeemed it generates a [`session`](#sessions)
for the device. A voucher is **available** while `activated_at IS NULL`; once
redeemed it is linked to a session and device.

**Used by:** [IVouchersApi](../api/voucher-api.md)

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Local voucher id. |
| `uuid` | VARCHAR | Stable voucher id. |
| `code` | VARCHAR(10) | The redeemable code (unique). |
| `provider_pkg` | VARCHAR(255) | Plugin package that issued the voucher. |
| `batch_uuid` | TEXT (nullable) | → `voucher_batches.uuid`; the batch it was generated in. |
| `session_type` | TEXT | `time` or `data`. |
| `time_secs` | INT | Time quota granted on redemption, seconds. |
| `data_mb` | INT | Data quota granted on redemption, MB. |
| `down_speed_mbps` / `up_speed_mbps` | INT | Speed caps applied to the generated session. |
| `session_exp_days` | INT (nullable) | Expiry (days) applied to the generated session. |
| `use_global` | INT | Whether the generated session uses the global pool. |
| `expires_at` | TIMESTAMP (nullable) | Voucher expiry — unredeemable after this (`NULL` = never expires). |
| `activated_at` | TIMESTAMP (nullable) | When redeemed. `NULL` = still available. |
| `session_id` | INTEGER → `sessions.id` | Session created on redemption (`SET NULL` on delete). |
| `device_id` | INTEGER → `devices.id` | Device that redeemed it (`SET NULL` on delete). |
| `created_at` | TIMESTAMP | When generated. |

!!! info "Availability check"
    A voucher is redeemable when
    `activated_at IS NULL AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)`.

### `voucher_batches`

A **group of vouchers** generated together (e.g. "print 100 one-hour vouchers").
Used for bulk management and reporting.

**Used by:** [IVouchersApi](../api/voucher-api.md)

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Row id. |
| `uuid` | TEXT | Stable batch id (unique); referenced by `vouchers.batch_uuid`. |
| `provider_pkg` | TEXT | Plugin package that created the batch. |
| `amount` | REAL (nullable) | Optional monetary value associated with the batch. |
| `metadata` | TEXT (nullable) | Optional JSON metadata. |
| `created_at` / `updated_at` | DATETIME | Row lifecycle. |

---

## Commerce

### `purchases`

An **in-app purchase order** — a checkout the device initiates (e.g. a plugin
purchase). A purchase is settled by one or more [`payments`](#payments).
Confirmation/cancellation are tracked via timestamps.

**Used by:** [IPurchaseRequest](../api/purchase-request.md) · [IPaymentsApi](../api/payments-api.md) · [IInAppPurchasesApi](../api/inapp-purchases-api.md) · [Accept Payments](../tutorials/accepting-payments.md)

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Local purchase id. |
| `uuid` | VARCHAR(36) | Stable purchase id (unique). |
| `device_id` | INTEGER → `devices.id` (nullable) | Buying device (`SET NULL`/nullable — purchase may outlive the device). |
| `sku` | VARCHAR(255) | Product SKU. |
| `name` | VARCHAR(255) | Product display name. |
| `description` | TEXT | Product description. |
| `price` | DECIMAL | Charged price. |
| `any_price` | BOOLEAN | `TRUE` if the buyer chooses the amount (pay-what-you-want). |
| `callback_plugin` | VARCHAR(255) | Plugin to notify on completion. |
| `callback_route` | VARCHAR(510) | Route the buyer returns to after payment. |
| `webhook_route` | VARCHAR(510) | Route for gateway webhooks. |
| `metadata` | TEXT | JSON blob of purchase context (default `{}`). |
| `processing` | BOOLEAN | `TRUE` while a payment is in flight. |
| `payment_url` | TEXT | Gateway checkout URL. |
| `payment_note` | TEXT | Free-text note about the payment. |
| `confirmed_at` | TIMESTAMP (nullable) | When the purchase was confirmed/paid. |
| `cancelled_at` | TIMESTAMP (nullable) | When cancelled. |
| `cancelled_reason` | TEXT | Why it was cancelled. |
| `created_at` | TIMESTAMP | When created. |

### `payments`

A **payment record** settling a [`purchase`](#purchases). One purchase may have
multiple payments.

**Used by:** [IPaymentsApi](../api/payments-api.md) · [IPurchaseRequest](../api/purchase-request.md) · [Accept Payments](../tutorials/accepting-payments.md)

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Row id. |
| `uuid` | VARCHAR(36) | Stable payment id (unique when non-empty). |
| `purchase_id` | INTEGER → `purchases.id` | Settled purchase (cascade delete). |
| `amount` | DECIMAL | Amount paid. |
| `provider` | VARCHAR | Plugin package that processed it. |
| `payment_method` | VARCHAR(255) | Method label within the provider (e.g. "Coins", a coinslot's alias). |
| `created_at` | TIMESTAMP | When recorded. |

---

## System

### `notifications`

Admin dashboard **notifications** (alerts, info messages).

**Used by:** [INotification](../api/notification.md)

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Row id. |
| `subject` | VARCHAR(255) | Notification title. |
| `content` | TEXT | Body text. |
| `status` | INTEGER | `0` Unread (default), `1` Read. |
| `type` | VARCHAR(100) | Severity — `info` (default), `success`, `warning`, `error`. |
| `created_at` / `updated_at` | TIMESTAMP | Row lifecycle. |
