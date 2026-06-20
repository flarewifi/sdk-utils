# IEventsApi

The `IEventsApi` is the unified event subscription API for plugins. It provides a single access point for reacting to session lifecycle changes, client device events, purchase outcomes, and voucher operations — replacing the individual `OnSessionEvent`, `OnClientEvent`, and related methods that were previously scattered across other APIs (those are now deprecated).

All registration methods are safe to call concurrently. **Dispatch is synchronous:** when an event fires, its callbacks run sequentially in the emitter's goroutine, in registration order. Deciding whether to run async is the handler's responsibility — a callback that must not block the emitting operation (or that runs long) should spawn its own goroutine.

Most events ignore the value a callback returns. A few let a callback **cancel the operation** by returning an error: `OnVoucherBeforeCreate` (cancels voucher creation) and `EventClientBeforeConnect` via `OnClientEvent` (cancels a client connection).

Access the `IEventsApi` via `api.Events()`:

```go
func Init(api sdkapi.IPluginApi) error {
    api.Events().OnSessionEvent(sdkapi.EventSessionCreated, func(data sdkapi.SessionEventData) error {
        // react to new session
        return nil
    })
    return nil
}
```

## IEventsApi Methods

### OnSessionEvent

Registers a callback that fires whenever the given session event occurs. The callback runs synchronously in the emitter's goroutine; its returned error is ignored by the emitter. Spawn a goroutine if it must not block the session operation.

**Available events:** `sdkapi.EventSessionCreated`, `sdkapi.EventSessionConnected`, `sdkapi.EventSessionDisconnected`, `sdkapi.EventSessionConsumed`, `sdkapi.EventSessionChanged`, `sdkapi.EventSessionDeleted`.

```go
api.Events().OnSessionEvent(sdkapi.EventSessionCreated, func(data sdkapi.SessionEventData) error {
    session := data.Session
    log.Printf("Session %s created for device ID %d", session.UUID(), session.DeviceID())
    return nil
})
```

The callback receives a [`SessionEventData`](#sessioneventdata-structure) struct containing the session and details about which fields changed (for `EventSessionChanged`).

### OnSessionBatchEvent

Registers a callback that fires whenever a batch of sessions is persisted to the database at once. The callback runs synchronously in the emitter's goroutine and receives a slice of all sessions that were saved in the batch; spawn a goroutine if it must not block.

The batch save system coalesces individual periodic session saves into a single database transaction every 60 seconds, reducing SQLite lock contention when many sessions are running simultaneously.

**Available events:** `sdkapi.EventSessionBatchUpdated`.

```go
api.Events().OnSessionBatchEvent(sdkapi.EventSessionBatchUpdated, func(sessions []sdkapi.IClientSession) error {
    api.Logger().Info("Batch saved %d sessions", len(sessions))
    for _, s := range sessions {
        api.Logger().Info("  Session %d: consumed %d secs, %.2f MB",
            s.ID(), s.ConsumedTimeSecs(), s.DataConsumption())
    }
    return nil
})
```

This event fires for periodic consumption saves only — not for session start, stop, or user-initiated changes. Use this event when you need to efficiently process multiple session updates in bulk.

### OnClientEvent

Registers a callback that fires whenever the given client device event occurs. The callback runs synchronously in the emitter's goroutine; spawn a goroutine if it must not block. For most client events the returned error is ignored — the exception is `EventClientBeforeConnect` (see below).

**Available events:**

| Event | Constant | Description |
|-------|----------|-------------|
| `"client:created"` | `sdkapi.EventClientCreated` | A new device record is created |
| `"client:registered"` | `sdkapi.EventClientRegistered` | A device completes registration |
| `"client:updated"` | `sdkapi.EventClientUpdated` | A device record is updated |
| `"client:connected"` | `sdkapi.EventClientConnected` | A device establishes a session |
| `"client:disconnected"` | `sdkapi.EventClientDisconnected` | A device disconnects |
| `"client:active"` | `sdkapi.EventClientActive` | Network activity detected at layer 3 (RFC 8908 captive portal probe) |
| `"client:before_connect"` | `sdkapi.EventClientBeforeConnect` | **Can cancel the connection** — fired before a device is connected; returning an error stops it |

```go
api.Events().OnClientEvent(sdkapi.EventClientConnected, func(ctx context.Context, clnt sdkapi.IClientDevice) error {
    log.Printf("Device %s connected with IPv4=%s", clnt.MacAddr(), clnt.Ipv4Addr())
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

### OnClientMerge

Registers a callback that fires after two device records have been successfully merged. The callback runs synchronously in the emitter's goroutine; spawn a goroutine if it must not block.

This event fires from multiple sources:

- **Real-time**: MAC-collision detected during device registration
- **Scheduled**: Background duplicate-device merge job
- **Plugin-triggered**: Calls to `api.SessionsMgr().MergeClientDevices()`

When the callback is invoked, the source device has already been deleted. The data includes the surviving device and the ID/UUID of the deleted device.

```go
api.Events().OnClientMerge(func(data sdkapi.EventClientMergeData) error {
    target := data.Target
    log.Printf("Device %s (%d) merged into %s (%d)",
        data.SourceDeviceUUID, data.SourceDeviceID,
        target.UUID(), target.ID())
    return nil
})
```

### OnPurchaseEvent

Registers a callback that fires whenever the given purchase event occurs. The callback runs synchronously in the emitter's goroutine; spawn a goroutine if it must not block.

**Available events:**

| Event | Constant | Description |
|-------|----------|-------------|
| `"purchase:success"` | `sdkapi.EventPurchaseSuccess` | Purchase confirmed successfully |
| `"purchase:failed"` | `sdkapi.EventPurchaseFailed` | Purchase confirmation or execution failed |
| `"purchase:cancelled"` | `sdkapi.EventPurchaseCancelled` | Purchase cancelled by the user |

```go
api.Events().OnPurchaseEvent(sdkapi.EventPurchaseSuccess, func(data sdkapi.PurchaseEventData) error {
    log.Printf("Purchase %d completed: %.2f", data.Purchase.ID(), data.Purchase.Price())
    return nil
})
```

### OnVoucherEvent

Registers a callback that fires whenever a single-voucher lifecycle event occurs. The callback runs synchronously in the emitter's goroutine; spawn a goroutine if it must not block.

**Available events:**

| Event | Constant | Description |
|-------|----------|-------------|
| `"voucher:activated"` | `sdkapi.EventVoucherActivated` | Voucher used to start a session |
| `"voucher:updated"` | `sdkapi.EventVoucherUpdated` | Voucher validity updated |
| `"voucher:deleted"` | `sdkapi.EventVoucherDeleted` | Voucher deleted |

```go
api.Events().OnVoucherEvent(sdkapi.EventVoucherActivated, func(v sdkapi.IVoucher) error {
    log.Printf("Voucher %s activated for session %d", v.Code(), v.Session().ID())
    return nil
})
```

### OnVoucherBatchEvent

Registers a callback that fires whenever a voucher-batch event occurs. The callback runs synchronously in the emitter's goroutine; spawn a goroutine if it must not block.

**Available events:**

| Event | Constant | Description |
|-------|----------|-------------|
| `"voucher:generated"` | `sdkapi.EventVoucherGenerated` | A batch of vouchers is created |
| `"voucher:batch_deleted"` | `sdkapi.EventVoucherBatchDeleted` | A voucher batch is deleted |

```go
api.Events().OnVoucherBatchEvent(sdkapi.EventVoucherGenerated, func(batch sdkapi.IVoucherBatch) error {
    log.Printf("Voucher batch %s generated with code length %d", batch.BatchUUID(), batch.CodeLength())
    return nil
})
```

### OnVoucherBeforeCreate

Registers a hook that is called before a batch of vouchers is created. Like all events the hooks run synchronously, in registration order, in the `CreateVouchers` caller's goroutine; the first hook that returns a non-nil error cancels voucher creation (the error is returned to the caller of `CreateVouchers`). Because it runs before any database writes, cancelling needs no rollback.

The hook receives a pointer to the creation params and may modify them (e.g., to enforce quota limits or override defaults). `BatchUUID` and the bandwidth defaults are already populated by the time hooks run, so credit/quota checks can rely on the final `BatchUUID`, `Count`, and `Amount`.

```go
api.Events().OnVoucherBeforeCreate(func(ctx context.Context, params *sdkapi.CreateVouchersParams) error {
    maxVouchers := 100
    if params.Count > maxVouchers {
        return fmt.Errorf("cannot create more than %d vouchers at once", maxVouchers)
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

The data received by `OnClientMerge` callbacks.

```go
type EventClientMergeData struct {
    Target           IClientDevice // Surviving device after merge
    SourceDeviceID   int64         // Database ID of the deleted device
    SourceDeviceUUID string        // UUID of the deleted device
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
| `sdkapi.EventSessionCreated` | `"session:created"` |
| `sdkapi.EventSessionConnected` | `"session:connected"` |
| `sdkapi.EventSessionDisconnected` | `"session:disconnected"` |
| `sdkapi.EventSessionConsumed` | `"session:expired"` |
| `sdkapi.EventSessionChanged` | `"session:changed"` |
| `sdkapi.EventSessionDeleted` | `"session:deleted"` |
| `sdkapi.EventSessionBatchUpdated` | `"session:batch-updated"` |

### ClientEvent Constants

| Constant | Value |
|----------|-------|
| `sdkapi.EventClientCreated` | `"client:created"` |
| `sdkapi.EventClientRegistered` | `"client:registered"` |
| `sdkapi.EventClientUpdated` | `"client:updated"` |
| `sdkapi.EventClientConnected` | `"client:connected"` |
| `sdkapi.EventClientDisconnected` | `"client:disconnected"` |
| `sdkapi.EventClientActive` | `"client:active"` |
| `sdkapi.EventClientBeforeConnect` | `"client:before_connect"` |

### PurchaseEvent Constants

| Constant | Value |
|----------|-------|
| `sdkapi.EventPurchaseSuccess` | `"purchase:success"` |
| `sdkapi.EventPurchaseFailed` | `"purchase:failed"` |
| `sdkapi.EventPurchaseCancelled` | `"purchase:cancelled"` |

### VoucherEvent Constants

| Constant | Value |
|----------|-------|
| `sdkapi.EventVoucherGenerated` | `"voucher:generated"` |
| `sdkapi.EventVoucherActivated` | `"voucher:activated"` |
| `sdkapi.EventVoucherUpdated` | `"voucher:updated"` |
| `sdkapi.EventVoucherDeleted` | `"voucher:deleted"` |
| `sdkapi.EventVoucherBatchDeleted` | `"voucher:batch_deleted"` |

### InternetEvent Constants

| Constant | Value |
|----------|-------|
| `sdkapi.EventInternetUp` | `"internet:up"` |
| `sdkapi.EventInternetDown` | `"internet:down"` |

### CreateVouchersParams

The parameters passed to `OnVoucherBeforeCreate` hooks. The hook may modify these values.

```go
type CreateVouchersParams struct {
    Count          int
    Type           SessionType
    TimeSecs       int64
    DataMb         int64
    DownSpeedMbps  int64      // default 10 Mbps if 0
    UpSpeedMbps    int64      // default 10 Mbps if 0
    SessionExpDays *int       // nil means session never expires
    UseGlobal      bool
    ExpiresAt      *time.Time // nil means voucher never expires
    BatchUUID      string
    Amount         *float64
}
```
