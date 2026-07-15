# IEventsApi

The `IEventsApi` is the unified event subscription API for plugins. It provides a single access point for reacting to session lifecycle changes, client device events, purchase outcomes, and voucher operations — replacing the individual `OnSessionEvent`, `OnClientEvent`, and related methods that were previously scattered across other APIs (those are now deprecated).

All registration methods are safe to call concurrently. **Dispatch is synchronous:** when an event fires, its callbacks run sequentially in the emitter's goroutine, in registration order. Deciding whether to run async is the handler's responsibility — a callback that must not block the emitting operation (or that runs long) should spawn its own goroutine.

Most events ignore the value a callback returns. The **"before" events**, however, let a callback **cancel the pending operation** by returning a non-nil error — the operation checks the first error, aborts, and propagates it to the caller. The cancellable hooks are:

| Method | Cancellable events |
|--------|--------------------|
| `OnClientEvent` | `EventClientBeforeConnect`, `EventClientBeforeCreate`, `EventClientBeforeUpdate`, `EventClientBeforeDisconnect` |
| `OnClientBatchEvent` | `EventClientBatchBeforeCreate` |
| `OnClientBeforeMerge` | `EventClientBeforeMerge` |
| `OnSessionEvent` | `EventSessionBeforeCreate`, `EventSessionBeforeConsume`, `EventSessionBeforeDelete` |
| `OnSessionBatchEvent` | `EventSessionBatchBeforeDelete`, `EventSessionBatchBeforeCreate` |
| `OnPurchaseEvent` | `EventPurchaseBeforeRequest`, `EventPurchaseBeforeCancel` |
| `OnVoucherEvent` | `EventVoucherBeforeCreate` (cancels the batch before any DB writes), `EventVoucherBeforeActivate` |
| `OnVoucherBatchEvent` | `EventVoucherBatchBeforeCreate`, `EventVoucherBatchBeforeDelete` |

Every "before" event that carries a not-yet-persisted record (create/request) delivers an **in-memory preview** (`ID == 0`), so cancelling never needs a rollback.

Access the `IEventsApi` via `api.Events()`:

```go
func Init(api sdkapi.IPluginApi) error {
    api.Events().OnSessionEvent(sdkapi.EventSessionCreated, func(ctx context.Context, data sdkapi.SessionEventData) error {
        // react to new session
        return nil
    })
    return nil
}
```

## IEventsApi Methods

### OnSessionEvent

Registers a callback that fires whenever the given session event occurs. The callback runs synchronously in the emitter's goroutine. For the terminal events its returned error is ignored; for the **"before" events** (`EventSessionBeforeCreate`, `EventSessionBeforeConsume`, `EventSessionBeforeDelete`) a non-nil error **cancels the operation** (see below). Spawn a goroutine if it must not block the session operation.

**Available events:** `sdkapi.EventSessionBeforeCreate`, `sdkapi.EventSessionCreated`, `sdkapi.EventSessionConnected`, `sdkapi.EventSessionDisconnected`, `sdkapi.EventSessionBeforeConsume`, `sdkapi.EventSessionConsumed`, `sdkapi.EventSessionChanged`, `sdkapi.EventSessionBeforeDelete`, `sdkapi.EventSessionDeleted`.

- **`EventSessionBeforeCreate`** — fires in `CreateSession()` before the INSERT with an in-memory preview session (`ID == 0`). Returning an error cancels creation.
- **`EventSessionBeforeConsume`** — fires when a running session is about to be finalized as consumed (time/data exhausted), while it is still running with its timer intact. Returning an error **vetoes consumption**: the session keeps running so a plugin can top it up (the plugin is then responsible for extending the session and re-arming enforcement).
- **`EventSessionBeforeDelete`** — fires in `DeleteSession()` before the device is disconnected or the row removed. Returning an error cancels the deletion.

```go
api.Events().OnSessionEvent(sdkapi.EventSessionCreated, func(ctx context.Context, data sdkapi.SessionEventData) error {
    session := data.Session
    log.Printf("Session %s created for device ID %d", session.UUID(), session.DeviceID())
    return nil
})
```

The callback receives a [`SessionEventData`](#sessioneventdata) struct containing the session and details about which fields changed (for `EventSessionChanged`).

### OnSessionBatchEvent

Registers a callback that fires whenever a batch of sessions is updated or about to be deleted at once. The callback runs synchronously in the emitter's goroutine and receives a slice of all sessions in the batch; spawn a goroutine if it must not block.

The batch save system coalesces individual periodic session saves into a single database transaction every 60 seconds, reducing database lock contention when many sessions are running simultaneously.

**Available events:**

| Event | Constant | Description |
|-------|----------|-------------|
| `"session:batch-updated"` | `sdkapi.EventSessionBatchUpdated` | A batch of periodic consumption saves was persisted |
| `"session:batch_before_delete"` | `sdkapi.EventSessionBatchBeforeDelete` | **Can cancel the batch delete** — fires once in `DeleteSessions()` before any session is removed; returning an error aborts the whole batch |
| `"session:batch_before_create"` | `sdkapi.EventSessionBatchBeforeCreate` | **Can cancel the batch create** — fires once in `CreateSessions()` before any row is inserted, with in-memory preview sessions (`ID == 0`); returning an error aborts the whole batch |
| `"session:batch_created"` | `sdkapi.EventSessionBatchCreated` | Fires once after a batch of sessions is successfully created, with the full list of created sessions |

```go
api.Events().OnSessionBatchEvent(sdkapi.EventSessionBatchUpdated, func(ctx context.Context, sessions []sdkapi.IClientSession) error {
    api.Logger().Info(fmt.Sprintf("Batch saved %d sessions", len(sessions)))
    return nil
})
```

`EventSessionBatchUpdated` fires for periodic consumption saves only — not for session start, stop, or user-initiated changes. `EventSessionBatchBeforeDelete` fires from the bulk [`DeleteSessions()`](./sessions-mgr-api.md) path and is the single cancellation point for bulk deletes (the per-session `EventSessionBeforeDelete` is not fired per item in that path). Likewise, `EventSessionBatchBeforeCreate`/`EventSessionBatchCreated` fire from the bulk [`CreateSessions()`](./sessions-mgr-api.md#createsessions) path — the per-session `EventSessionBeforeCreate` is not fired per item there, but the per-session `EventSessionCreated` (via `OnSessionEvent`) still fires once for each session, after the whole batch has been inserted successfully.

### OnClientEvent

Registers a callback that fires whenever the given client device event occurs. The callback runs synchronously in the emitter's goroutine; spawn a goroutine if it must not block. For the terminal client events the returned error is ignored — the exceptions are the **"before" events** (`EventClientBeforeConnect`, `EventClientBeforeCreate`, `EventClientBeforeUpdate`, `EventClientBeforeDisconnect`), which can cancel the operation (see below). Merges are cancelled via [`OnClientBeforeMerge`](#onclientbeforemerge), not this method.

**Available events:**

| Event | Constant | Description |
|-------|----------|-------------|
| `"client:before_create"` | `sdkapi.EventClientBeforeCreate` | **Can cancel registration** — fires before a new device row is inserted, with an in-memory preview (`ID == 0`) |
| `"client:created"` | `sdkapi.EventClientCreated` | A new device record is created |
| `"client:registered"` | `sdkapi.EventClientRegistered` | A device completes registration |
| `"client:before_update"` | `sdkapi.EventClientBeforeUpdate` | **Can cancel the update** — fires before a device's network details are written (and before any reconnect) |
| `"client:updated"` | `sdkapi.EventClientUpdated` | A device record is updated |
| `"client:connected"` | `sdkapi.EventClientConnected` | A device establishes a session |
| `"client:before_disconnect"` | `sdkapi.EventClientBeforeDisconnect` | **Can cancel the disconnect** — fires before any teardown, for explicit disconnects only |
| `"client:disconnected"` | `sdkapi.EventClientDisconnected` | A device disconnects |
| `"client:active"` | `sdkapi.EventClientActive` | Network activity detected at layer 3 (RFC 8908 captive portal probe) |
| `"client:before_connect"` | `sdkapi.EventClientBeforeConnect` | **Can cancel the connection** — fired before a device is connected; returning an error stops it |

```go
api.Events().OnClientEvent(sdkapi.EventClientConnected, func(ctx context.Context, clnt sdkapi.IClientDevice) error {
    log.Printf("Device %s connected", clnt.MacAddr())
    return nil
})
```

#### EventClientBeforeConnect (can cancel the connection)

`EventClientBeforeConnect` can cancel a connection. Like all events its callbacks run synchronously, in registration order, inside the `Connect()` caller's goroutine — what sets it apart is that `Connect()` honors the returned error. If a callback returns a non-nil error, the connection is **not** established and the error is returned to the caller of `Connect()`. Because the hook fires before any side effects (firewall rules, session start), cancelling needs no rollback.

Use it for last-mile policy enforcement — quota/credit checks, time-of-day rules, device allow/deny lists, etc. Keep callbacks fast: they block the connection while they run (each is bounded by an internal timeout, but a slow hook delays the user's connection).

```go
api.Events().OnClientEvent(sdkapi.EventClientBeforeConnect, func(ctx context.Context, clnt sdkapi.IClientDevice) error {
    if overQuota(clnt) {
        // Returning an error here aborts the connection; the message
        // surfaces to the caller of Connect().
        return errors.New("data quota exceeded")
    }
    return nil
})
```

### OnClientBeforeMerge

Registers a callback that fires **before** two device records are merged, while **both still exist**. The callback runs synchronously in the emitter's goroutine. Returning a non-nil error **cancels the merge** before any data is transferred or deleted.

The [`EventClientMergeData`](#eventclientmergedata) here carries both devices: `Target` (the survivor) and `Source` (the device about to be deleted) — both non-nil. This is where you veto an unwanted merge, or notify an external system before the source disappears.

Cancellation behavior depends on the merge's origin:

- From an **explicit** `api.SessionsMgr().MergeClientDevices()` call, the error propagates back to the caller.
- From an **implicit** MAC-collision merge during registration, the merge is simply skipped (the colliding MAC is freed so the update can still proceed) — nothing fails.

```go
api.Events().OnClientBeforeMerge(func(ctx context.Context, data sdkapi.EventClientMergeData) error {
    if isProtected(data.Source) {
        return fmt.Errorf("device %s is protected from merging", data.Source.UUID())
    }
    return nil
})
```

### OnClientMerge

Registers a callback that fires after two device records have been successfully merged. The callback runs synchronously in the emitter's goroutine; spawn a goroutine if it must not block.

This event fires from multiple sources:

- **Real-time**: MAC-collision detected during device registration
- **Scheduled**: Background duplicate-device merge job
- **Plugin-triggered**: Calls to `api.SessionsMgr().MergeClientDevices()`

When the callback is invoked, the source device has **already been deleted** — so `data.Source` is `nil` here; use `data.SourceDeviceID`/`data.SourceDeviceUUID` (captured before deletion) instead. `data.Target` is the surviving device that received all transferred data.

```go
api.Events().OnClientMerge(func(ctx context.Context, data sdkapi.EventClientMergeData) error {
    target := data.Target
    log.Printf("Device %s (%d) merged into %s (%d)",
        data.SourceDeviceUUID, data.SourceDeviceID,
        target.UUID(), target.ID())
    return nil
})
```

### OnClientBatchEvent

Registers a callback that fires whenever a batch of client devices is registered at once, from [`BatchRegisterClient()`](./clients-mgr-api.md#batchregisterclient). The callback runs synchronously in the emitter's goroutine and receives a slice of all devices in the batch; spawn a goroutine if it must not block.

**Available events:**

| Event | Constant | Description |
|-------|----------|-------------|
| `"client:batch_before_create"` | `sdkapi.EventClientBatchBeforeCreate` | **Can cancel the batch** — fires once before any DB writes, with the in-memory device previews passed to `BatchRegisterClient()` (`ID == 0`). Returning an error cancels the whole batch. |
| `"client:batch_created"` | `sdkapi.EventClientBatchCreated` | Fires once after the whole batch is successfully committed, with the full list of created devices |

```go
api.Events().OnClientBatchEvent(sdkapi.EventClientBatchCreated, func(ctx context.Context, clients []sdkapi.IClientDevice) error {
    api.Logger().Info(fmt.Sprintf("Registered %d devices", len(clients)))
    return nil
})
```

The per-device `EventClientBeforeCreate` (via `OnClientEvent`) also still fires once for each device — but, notably, **before** the creation transaction opens, not inside it. This app runs SQLite through a single shared connection (`db.SetMaxOpenConns(1)`), so a subscriber's own DB call made while a transaction is open would block forever waiting for a connection only that same (blocked) call could free. Firing all per-device checks first means a subscriber's query is always safe, and a veto there needs no rollback — nothing has been written yet. Only once every device has passed does the batch open its transaction and insert. The per-device `EventClientCreated`/`EventClientRegistered` fire once for each device, after the whole batch has committed successfully.

### OnPurchaseEvent

Registers a callback that fires whenever the given purchase event occurs. The callback runs synchronously in the emitter's goroutine; spawn a goroutine if it must not block. The **"before" events** (`EventPurchaseBeforeRequest`, `EventPurchaseBeforeCancel`) can cancel the operation by returning an error.

**Available events:**

| Event | Constant | Description |
|-------|----------|-------------|
| `"purchase:before_request"` | `sdkapi.EventPurchaseBeforeRequest` | **Can cancel the request** — fires before a purchase request is created, with an in-memory preview purchase (`ID == 0`). Use for eligibility/quota/block-list checks |
| `"purchase:success"` | `sdkapi.EventPurchaseSuccess` | Purchase confirmed successfully |
| `"purchase:failed"` | `sdkapi.EventPurchaseFailed` | Purchase confirmation or execution failed |
| `"purchase:before_cancelled"` | `sdkapi.EventPurchaseBeforeCancel` | **Can cancel the cancellation** — fires before a purchase is cancelled |
| `"purchase:cancelled"` | `sdkapi.EventPurchaseCancelled` | Purchase cancelled by the user |

```go
api.Events().OnPurchaseEvent(sdkapi.EventPurchaseSuccess, func(ctx context.Context, data sdkapi.PurchaseEventData) error {
    log.Printf("Purchase %d completed: %.2f", data.Purchase.ID(), data.Purchase.Price())
    return nil
})

// Admission control before a purchase request is created:
api.Events().OnPurchaseEvent(sdkapi.EventPurchaseBeforeRequest, func(ctx context.Context, data sdkapi.PurchaseEventData) error {
    if data.Device != nil && isBlocked(data.Device) {
        return fmt.Errorf("device is not allowed to make purchases")
    }
    return nil
})
```

> For `EventPurchaseBeforeRequest`, `data.Purchase` is an in-memory preview: `ID()` is 0 and `UUID()` is empty, but `Sku()`, `Name()`, `Price()`, `Description()`, and `Metadata()` carry the pending request's values. `data.Device` is the buyer (nil for admin/device-less purchases).

### OnVoucherEvent

Registers a callback that fires whenever a single-voucher lifecycle event occurs. The callback runs synchronously in the emitter's goroutine; spawn a goroutine if it must not block.

**Available events:**

| Event | Constant | Description |
|-------|----------|-------------|
| `"voucher:before_create"` | `sdkapi.EventVoucherBeforeCreate` | **Can cancel creation** — fires once for each voucher, BEFORE the creation transaction opens. The voucher is an in-memory preview (`ID == 0`). Returning an error cancels the whole batch before any row is inserted — no rollback needed. |
| `"voucher:before_activate"` | `sdkapi.EventVoucherBeforeActivate` | **Can cancel activation** — fires before a voucher is activated (before any session is created); returning an error aborts it |
| `"voucher:activated"` | `sdkapi.EventVoucherActivated` | Voucher used to start a session |
| `"voucher:updated"` | `sdkapi.EventVoucherUpdated` | Voucher validity updated |
| `"voucher:deleted"` | `sdkapi.EventVoucherDeleted` | Voucher deleted |

```go
api.Events().OnVoucherEvent(sdkapi.EventVoucherActivated, func(ctx context.Context, v sdkapi.IVoucher) error {
    log.Printf("Voucher %s activated for session %d", v.Code(), v.Session().ID())
    return nil
})
```

#### EventVoucherBeforeCreate (per-voucher, can cancel)

Fires once for each individual voucher in the batch, **before** the creation transaction opens — not inside it. The voucher object is an in-memory preview — `ID()` is 0, `Session()` and `Device()` are nil, but `Code()`, `UUID()`, `BatchUUID()`, and all param-derived fields (`Type()`, `TimeSecs()`, `DataMb()`, etc.) are set. Returning an error cancels the whole batch before any row is inserted — no rollback needed, since nothing has been written yet.

> **Why before, not inside, the transaction:** this app runs SQLite through a single shared connection (`db.SetMaxOpenConns(1)`). If this event fired from inside the creation transaction, a subscriber making its own DB call from this callback would block forever waiting for a connection — the only one in the pool is checked out by the very transaction this call is blocking. Firing before the transaction opens means a subscriber's query here is always safe.

```go
api.Events().OnVoucherEvent(sdkapi.EventVoucherBeforeCreate, func(ctx context.Context, v sdkapi.IVoucher) error {
    if v.TimeSecs() < 300 {
        return fmt.Errorf("voucher time must be at least 5 minutes")
    }
    return nil
})
```

### OnVoucherBatchEvent

Registers a callback that fires whenever a voucher-batch event occurs. The callback runs synchronously in the emitter's goroutine; spawn a goroutine if it must not block.

**Available events:**

| Event | Constant | Description |
|-------|----------|-------------|
| `"voucher:before_create"` | `sdkapi.EventVoucherBatchBeforeCreate` | **Can cancel creation** — fires once before any DB writes, with an in-memory preview batch (`ID == 0`). Returning an error cancels the whole batch. |
| `"voucher:batch_created"` | `sdkapi.EventVoucherBatchCreated` | A batch of vouchers was successfully created |
| `"voucher:batch_before_delete"` | `sdkapi.EventVoucherBatchBeforeDelete` | **Can cancel the delete** — fires before a voucher batch is deleted; returning an error aborts it |
| `"voucher:batch_deleted"` | `sdkapi.EventVoucherBatchDeleted` | A voucher batch is deleted |

> **Renamed:** `EventVoucherBatchCreated` was previously `EventVoucherGenerated` (`"voucher:generated"`). Update any subscriptions to the new constant.

```go
api.Events().OnVoucherBatchEvent(sdkapi.EventVoucherBatchCreated, func(ctx context.Context, batch sdkapi.IVoucherBatch) error {
    log.Printf("Voucher batch %s created with %d vouchers", batch.UUID(), batch.VouchersCount())
    return nil
})
```

#### EventVoucherBeforeCreate (batch-level, can cancel)

Fires once before any database writes. The batch object is an in-memory preview — `ID()` is 0 (the voucher rows don't exist yet), but `UUID()`, `Amount()`, `VouchersCount()` (the *intended* count), and `ProviderPkg()` are all set. This is the right place for batch-level checks such as reseller credit validation, count limits, or quota enforcement. Returning an error cancels creation with no rollback needed.

```go
api.Events().OnVoucherBatchEvent(sdkapi.EventVoucherBatchBeforeCreate, func(ctx context.Context, batch sdkapi.IVoucherBatch) error {
    maxVouchers := 100
    if batch.VouchersCount() > int64(maxVouchers) {
        return fmt.Errorf("cannot create more than %d vouchers at once", maxVouchers)
    }
    // Credit check example
    if amount := batch.Amount(); amount != nil && *amount > 0 {
        return validateCredits(ctx, *amount, batch.VouchersCount(), batch.UUID())
    }
    return nil
})
```

### OnInternetEvent

Registers a callback that fires whenever the machine's internet connectivity changes, as observed by the core's **online monitor** — a background service that periodically probes for internet reachability. The callback runs synchronously in the monitor's goroutine, in registration order; its returned error is logged but does not stop other callbacks.

A callback that does slow work (downloads, package installs, API calls) **must spawn its own goroutine** so it does not stall the monitor's polling loop.

**Available events:**

| Event | Constant | Description |
|-------|----------|-------------|
| `"internet:up"` | `sdkapi.EventInternetUp` | The machine gained internet access — at boot once connectivity first arrives, or after an outage is restored |
| `"internet:down"` | `sdkapi.EventInternetDown` | Internet access was lost |

The core itself subscribes to `EventInternetUp` to run network-dependent install work — a plugin's `system_packages` (`opkg`) and its `preinstall`/`postinstall` scripts — so a machine that was flashed offline is still fully provisioned the moment it reaches the internet. See [`plugin.json`](./plugin.json.md) for how that provisioning works.

```go
api.Events().OnInternetEvent(sdkapi.EventInternetUp, func(ctx context.Context) error {
    // Slow work must not block the monitor — run it in a goroutine.
    go func() {
        if err := syncPendingDataToCloud(); err != nil {
            api.Logger().Error("cloud sync failed: " + err.Error())
        }
    }()
    return nil
})

api.Events().OnInternetEvent(sdkapi.EventInternetDown, func(ctx context.Context) error {
    api.Logger().Info("machine went offline")
    return nil
})
```

> **Need the current status, not a change?** Use [`api.Machine().IsOnline()`](./machine-api.md#isonline) for a one-off check of whether the machine has internet right now — it reads the same online-monitor signal that drives these events. Subscribe with `OnInternetEvent` to *react* to transitions; call `IsOnline()` to *query* the state at the moment you need it (e.g. just before attempting a network call).

> **Emission starts after boot.** The online monitor begins probing — and therefore emits `EventInternetUp`/`EventInternetDown` — only once boot completes (`EventBoot`, below). During boot the WAN link is still coming up, so probing then would emit a spurious `EventInternetDown` and surface a false "no internet" notification on every reboot.

### OnBoot

Registers a callback that fires **once**, after the machine's boot sequence has fully completed — the captive portal is up, the network is initialized, and provisioning has run or been deferred. The callback runs synchronously in the boot goroutine, in registration order; its returned error is logged but does not stop other callbacks. Spawn a goroutine if it must do slow work, so it does not stall boot finalization.

**Available events:**

| Event | Constant | Description |
|-------|----------|-------------|
| `"boot:complete"` | `sdkapi.EventBoot` | The boot sequence finished; the machine is fully up |

Use it to start work that should only begin once the machine is fully booted (the core uses it to gate the online monitor's connectivity emissions). A callback that starts a long-lived service must **not** retain the passed `ctx` — it is cancelled when the callback returns; capture `context.Background()` instead.

```go
api.Events().OnBoot(sdkapi.EventBoot, func(ctx context.Context) error {
    api.Logger().Info("machine fully booted")
    return nil
})
```

### OnDhcpEvent

Registers a callback that fires whenever dnsmasq reports a DHCPv4 lease event via its **dhcp-script** hook (see [OpenWrt's DHCP docs](https://openwrt.org/docs/guide-user/base-system/dhcp)). The callback runs synchronously in the core's DHCP listener goroutine, in registration order; its returned error is logged but does not stop other callbacks and **cannot veto the lease change** — it has already happened by the time dnsmasq calls the hook.

A callback that does slow work must spawn its own goroutine so it does not stall the listener.

> **IPv6 is not covered.** On this machine IPv6 leases are served by `odhcpd`, not dnsmasq, using a separate `leasetrigger` hook already wired to the stock OpenWrt hosts-file/NDP bookkeeping — `OnDhcpEvent` only ever fires for DHCPv4.

**Available events:**

| Event | Constant | Description |
|-------|----------|--------------|
| `"dhcp:lease_add"` | `sdkapi.EventDhcpLeaseAdd` | dnsmasq handed a brand-new lease to a client |
| `"dhcp:lease_old"` | `sdkapi.EventDhcpLeaseOld` | An existing lease was renewed/rebound, or replayed because dnsmasq itself started/reloaded |
| `"dhcp:lease_del"` | `sdkapi.EventDhcpLeaseDel` | A lease was destroyed — released by the client, expired, or removed administratively |

```go
api.Events().OnDhcpEvent(sdkapi.EventDhcpLeaseAdd, func(ctx context.Context, data sdkapi.DhcpEventData) error {
    api.Logger().Info(fmt.Sprintf("new DHCP lease: %s -> %s (%s)", data.Mac, data.Ip, data.Hostname))
    return nil
})
```

> `data.Hostname` is only populated for `EventDhcpLeaseAdd`, and for `EventDhcpLeaseOld` when a client actually resumed/renewed its lease — dnsmasq does not persist hostnames in its lease database, so a cold-restart replay of `EventDhcpLeaseOld` leaves it empty. `data.Interface` is likewise empty for that same restart-replay case. `data.LeaseExpires` is zero for `EventDhcpLeaseDel`.

## Supporting Types

### SessionEventData

The data received by `OnSessionEvent` callbacks.

```go
type SessionEventData struct {
    Session       IClientSession
    ChangedFields SessionChangedFields // Which fields changed (only set for EventSessionChanged)
}
```

`ChangedFields` is only populated for `EventSessionChanged` events. See [SessionChangedFields](./client-session.md#sessionchangedfields) for the full list of trackable fields.

### EventClientMergeData

The data received by `OnClientBeforeMerge` and `OnClientMerge` callbacks.

```go
type EventClientMergeData struct {
    Target           IClientDevice // Surviving device (before and after the merge)
    Source           IClientDevice // Device about to be deleted — set ONLY for the pre-merge
                                   // EventClientBeforeMerge; nil for the post-merge EventClientMerge
    SourceDeviceID   int64         // Database ID of the (to-be-)deleted device
    SourceDeviceUUID string        // UUID of the (to-be-)deleted device
}
```

`Source` is populated only for `OnClientBeforeMerge` (where the device still exists). In `OnClientMerge` the source row is already gone, so `Source` is `nil` — use `SourceDeviceID`/`SourceDeviceUUID` there.

### DhcpEventData

The data received by `OnDhcpEvent` callbacks — the lease details dnsmasq passed to its dhcp-script hook.

```go
type DhcpEventData struct {
    Mac          string    // Client hardware MAC address
    Ip           string    // Leased IPv4 address
    Hostname     string    // Hostname the client supplied, if any (see OnDhcpEvent notes above)
    Interface    string    // Interface the DHCP request arrived on (e.g. "br-lan")
    Tags         string    // dnsmasq config tags matched for this transaction, space-separated
    LeaseExpires time.Time // Lease expiry time; zero for EventDhcpLeaseDel
}
```

### PurchaseEventData

The data received by `OnPurchaseEvent` callbacks.

```go
type PurchaseEventData struct {
    Purchase IPurchaseRequest // The purchase that triggered the event
    Device   IClientDevice    // Device associated with the purchase
    Reason   string           // Failure/cancellation reason (empty for success)
}
```

### SessionEvent Constants

| Constant | Value |
|----------|-------|
| `sdkapi.EventSessionBeforeCreate` | `"session:before_create"` |
| `sdkapi.EventSessionCreated` | `"session:created"` |
| `sdkapi.EventSessionConnected` | `"session:connected"` |
| `sdkapi.EventSessionDisconnected` | `"session:disconnected"` |
| `sdkapi.EventSessionBeforeConsume` | `"session:before_consume"` |
| `sdkapi.EventSessionConsumed` | `"session:expired"` |
| `sdkapi.EventSessionChanged` | `"session:changed"` |
| `sdkapi.EventSessionBeforeDelete` | `"session:before_delete"` |
| `sdkapi.EventSessionDeleted` | `"session:deleted"` |
| `sdkapi.EventSessionBatchUpdated` | `"session:batch-updated"` |
| `sdkapi.EventSessionBatchBeforeDelete` | `"session:batch_before_delete"` |
| `sdkapi.EventSessionBatchBeforeCreate` | `"session:batch_before_create"` |
| `sdkapi.EventSessionBatchCreated` | `"session:batch_created"` |

### ClientEvent Constants

| Constant | Value |
|----------|-------|
| `sdkapi.EventClientBeforeCreate` | `"client:before_create"` |
| `sdkapi.EventClientCreated` | `"client:created"` |
| `sdkapi.EventClientRegistered` | `"client:registered"` |
| `sdkapi.EventClientBeforeUpdate` | `"client:before_update"` |
| `sdkapi.EventClientUpdated` | `"client:updated"` |
| `sdkapi.EventClientConnected` | `"client:connected"` |
| `sdkapi.EventClientBeforeDisconnect` | `"client:before_disconnect"` |
| `sdkapi.EventClientDisconnected` | `"client:disconnected"` |
| `sdkapi.EventClientActive` | `"client:active"` |
| `sdkapi.EventClientBeforeConnect` | `"client:before_connect"` |
| `sdkapi.EventClientBeforeMerge` | `"client:before_merge"` |
| `sdkapi.EventClientMerge` | `"client:merged"` |

### ClientBatchEvent Constants (batch-level — use with `OnClientBatchEvent`)

| Constant | Value | Notes |
|----------|-------|-------|
| `sdkapi.EventClientBatchBeforeCreate` | `"client:batch_before_create"` | Batch pre-create — returning an error cancels the whole batch before any row is inserted |
| `sdkapi.EventClientBatchCreated` | `"client:batch_created"` | Fires after a batch of devices is successfully registered |

### PurchaseEvent Constants

| Constant | Value |
|----------|-------|
| `sdkapi.EventPurchaseBeforeRequest` | `"purchase:before_request"` |
| `sdkapi.EventPurchaseSuccess` | `"purchase:success"` |
| `sdkapi.EventPurchaseFailed` | `"purchase:failed"` |
| `sdkapi.EventPurchaseBeforeCancel` | `"purchase:before_cancelled"` |
| `sdkapi.EventPurchaseCancelled` | `"purchase:cancelled"` |

### VoucherEvent Constants (single-voucher — use with `OnVoucherEvent`)

| Constant | Value | Notes |
|----------|-------|-------|
| `sdkapi.EventVoucherBeforeCreate` | `"voucher:before_create"` | Per-voucher pre-create, fires before the transaction opens — returning an error cancels the whole batch before any row is inserted |
| `sdkapi.EventVoucherBeforeActivate` | `"voucher:before_activate"` | Pre-activate — returning an error cancels activation |
| `sdkapi.EventVoucherActivated` | `"voucher:activated"` | |
| `sdkapi.EventVoucherUpdated` | `"voucher:updated"` | |
| `sdkapi.EventVoucherDeleted` | `"voucher:deleted"` | |

### VoucherBatchEvent Constants (batch-level — use with `OnVoucherBatchEvent`)

| Constant | Value | Notes |
|----------|-------|-------|
| `sdkapi.EventVoucherBatchBeforeCreate` | `"voucher:before_create"` | Batch pre-create — returning an error cancels creation before any DB writes |
| `sdkapi.EventVoucherBatchCreated` | `"voucher:batch_created"` | Fires after successful batch creation (renamed from `EventVoucherGenerated`) |
| `sdkapi.EventVoucherBatchBeforeDelete` | `"voucher:batch_before_delete"` | Batch pre-delete — returning an error cancels the deletion |
| `sdkapi.EventVoucherBatchDeleted` | `"voucher:batch_deleted"` | |

### PaymentEvent Constants

| Constant | Value |
|----------|-------|
| `sdkapi.EventPaymentOptionsChanged` | `"payment:options:changed"` |

### InternetEvent Constants

| Constant | Value |
|----------|-------|
| `sdkapi.EventInternetUp` | `"internet:up"` |
| `sdkapi.EventInternetDown` | `"internet:down"` |

### BootEvent Constants

| Constant | Value |
|----------|-------|
| `sdkapi.EventBoot` | `"boot:complete"` |

### DhcpEvent Constants

| Constant | Value |
|----------|-------|
| `sdkapi.EventDhcpLeaseAdd` | `"dhcp:lease_add"` |
| `sdkapi.EventDhcpLeaseOld` | `"dhcp:lease_old"` |
| `sdkapi.EventDhcpLeaseDel` | `"dhcp:lease_del"` |

