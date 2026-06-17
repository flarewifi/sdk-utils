# IEventsApi

The `IEventsApi` is the unified event subscription API for plugins. It provides a single access point for reacting to session lifecycle changes, client device events, purchase outcomes, and voucher operations — replacing the individual `OnSessionEvent`, `OnClientEvent`, and related methods that were previously scattered across other APIs (those are now deprecated).

All registration methods are safe to call concurrently. Callbacks are dispatched asynchronously (each in its own goroutine) except for `OnBeforeCreate`, which is synchronous and can block or abort voucher creation.

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

Registers a callback that fires whenever the given session event occurs. The callback runs asynchronously; errors are logged but not propagated to the caller that emitted the event.

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

Registers a callback that fires whenever a batch of sessions is persisted to the database at once. The callback runs asynchronously and receives a slice of all sessions that were saved in the batch.

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

Registers a callback that fires whenever the given client device event occurs. The callback runs asynchronously.

**Available events:**

| Event | Constant | Description |
|-------|----------|-------------|
| `"client:created"` | `sdkapi.EventClientCreated` | A new device record is created |
| `"client:registered"` | `sdkapi.EventClientRegistered` | A device completes registration |
| `"client:updated"` | `sdkapi.EventClientUpdated` | A device record is updated |
| `"client:connected"` | `sdkapi.EventClientConnected` | A device establishes a session |
| `"client:disconnected"` | `sdkapi.EventClientDisconnected` | A device disconnects |
| `"client:active"` | `sdkapi.EventClientActive` | Network activity detected at layer 3 (RFC 8908 captive portal probe) |

```go
api.Events().OnClientEvent(sdkapi.EventClientConnected, func(clnt sdkapi.IClientDevice) error {
    log.Printf("Device %s connected with IPv4=%s", clnt.MacAddr(), clnt.Ipv4Addr())
    return nil
})
```

### OnClientMerge

Registers a callback that fires after two device records have been successfully merged. The callback runs asynchronously.

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

Registers a callback that fires whenever the given purchase event occurs. The callback runs asynchronously.

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

Registers a callback that fires whenever a single-voucher lifecycle event occurs. The callback runs asynchronously.

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

Registers a callback that fires whenever a voucher-batch event occurs. The callback runs asynchronously.

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

### OnBeforeCreate

Registers a synchronous hook that is called before vouchers are created. Hooks run in registration order; the first hook that returns a non-nil error aborts the chain and prevents voucher creation.

The hook receives a pointer to the creation params and may modify them (e.g., to enforce quota limits or override defaults).

```go
api.Events().OnBeforeCreate(func(ctx context.Context, params *sdkapi.CreateVouchersParams) error {
    maxVouchers := 100
    if params.Count > maxVouchers {
        return fmt.Errorf("cannot create more than %d vouchers at once", maxVouchers)
    }
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

### CreateVouchersParams

The parameters passed to `OnBeforeCreate` hooks. The hook may modify these values.

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
