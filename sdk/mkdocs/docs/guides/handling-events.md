# Handling Events

The `IEventsApi` is the single access point for all plugin event subscriptions. Rather than calling event-registration methods scattered across different APIs (e.g., `api.SessionsMgr().OnSessionEvent(...)` — which no longer exists), every callback is registered through `api.Events()`.

```go
func Init(api sdkapi.IPluginApi) error {
    events := api.Events()

    events.OnSessionEvent(sdkapi.EventSessionCreated, func(ctx context.Context, data sdkapi.SessionEventData) error {
        // react to new session
        return nil
    })

    return nil
}
```

**Dispatch is synchronous.** When an event fires, its callbacks run sequentially in the emitter's goroutine, in registration order. If a callback does slow work (network calls, file I/O, database writes), it must spawn its own goroutine to avoid blocking the emitting operation.

See [IEventsApi](../api/events-api.md) for the complete API reference.

---

## Session Events

Register with `OnSessionEvent` to react to session lifecycle changes. The callback receives a [`SessionEventData`](../api/events-api.md#sessioneventdata) struct containing the session and, for `EventSessionChanged`, a field-change map.

```go
events := api.Events()

events.OnSessionEvent(sdkapi.EventSessionCreated, func(ctx context.Context, data sdkapi.SessionEventData) error {
    session := data.Session
    api.Logger().Info("Session created: " + session.UUID())
    return nil
})

events.OnSessionEvent(sdkapi.EventSessionDeleted, func(ctx context.Context, data sdkapi.SessionEventData) error {
    // Sync deletion to an external system — run async to avoid blocking
    go syncDeletion(data.Session.UUID())
    return nil
})
```

### Available Session Events

| Constant | When it fires |
|----------|---------------|
| `sdkapi.EventSessionCreated` | `CreateSession()` completes |
| `sdkapi.EventSessionConnected` | Device starts consuming a session |
| `sdkapi.EventSessionDisconnected` | Device disconnects; session is paused |
| `sdkapi.EventSessionConsumed` | Time or data fully exhausted |
| `sdkapi.EventSessionChanged` | Plugin/admin calls `session.Save()` after modifications |
| `sdkapi.EventSessionDeleted` | `DeleteSession()` removes a session |

### Session Batch Events

`OnSessionBatchEvent` fires after the periodic (≈60 s) batch-save flush, once per flush, with all [`IClientSession`](../api/client-session.md) records saved in that transaction. Use it for bulk cloud sync instead of processing N individual `EventSessionChanged` events.

```go
events.OnSessionBatchEvent(sdkapi.EventSessionBatchUpdated, func(ctx context.Context, sessions []sdkapi.IClientSession) error {
    go syncBatchToCloud(sessions)
    return nil
})
```

### Tracking Exactly What Changed

For `EventSessionChanged`, `data.ChangedFields` tells you which fields were modified:

```go
events.OnSessionEvent(sdkapi.EventSessionChanged, func(ctx context.Context, data sdkapi.SessionEventData) error {
    changed := data.ChangedFields
    session := data.Session

    if changed.TimeSecs || changed.TimeCons {
        api.Logger().Infof("Time updated for session %s", session.UUID())
    }
    if changed.DownMbits || changed.UpMbits {
        api.Logger().Infof("Bandwidth updated for session %s", session.UUID())
    }
    return nil
})
```

---

## Client Device Events

Register with `OnClientEvent` to react to [`IClientDevice`](../api/client-device.md) lifecycle changes.

```go
events.OnClientEvent(sdkapi.EventClientConnected, func(ctx context.Context, clnt sdkapi.IClientDevice) error {
    api.Logger().Info("Device connected: " + clnt.MacAddr())
    return nil
})

events.OnClientEvent(sdkapi.EventClientDisconnected, func(ctx context.Context, clnt sdkapi.IClientDevice) error {
    go syncDisconnect(clnt.UUID())
    return nil
})
```

### Available Client Events

| Constant | When it fires |
|----------|---------------|
| `sdkapi.EventClientCreated` | New device record created |
| `sdkapi.EventClientRegistered` | Device completes registration |
| `sdkapi.EventClientUpdated` | Device record updated |
| `sdkapi.EventClientConnected` | Device establishes a session |
| `sdkapi.EventClientDisconnected` | Device disconnects |
| `sdkapi.EventClientActive` | Layer-3 network activity detected |
| `sdkapi.EventClientBeforeConnect` | **Can cancel the connection** (see below) |

### Cancelling a Connection

`EventClientBeforeConnect` is the only client event whose return value matters. If a callback returns a non-nil error, the connection is aborted before any firewall rules are applied — no rollback is needed.

Use this for quota checks, time-of-day restrictions, or device allow/deny lists:

```go
events.OnClientEvent(sdkapi.EventClientBeforeConnect, func(ctx context.Context, clnt sdkapi.IClientDevice) error {
    if isDeviceBlocked(clnt.MacAddr()) {
        return errors.New("device is not allowed on this network")
    }
    return nil
})
```

### Device Merge Events

`OnClientMerge` fires after two device records are merged into one. The callback receives an [`EventClientMergeData`](../api/events-api.md#eventclientmergedata) value; the source device is already deleted when it runs.

```go
events.OnClientMerge(func(ctx context.Context, data sdkapi.EventClientMergeData) error {
    // Notify external system: source UUID was merged into target
    go syncMerge(data.SourceDeviceUUID, data.Target.UUID())
    return nil
})
```

---

## Internet Events

`OnInternetEvent` fires when the machine's internet reachability changes. The online monitor runs periodically in the background; the callback runs synchronously in the monitor's goroutine.

**Always spawn a goroutine** for any slow work here — slow callbacks delay the monitor's next poll cycle.

```go
events.OnInternetEvent(sdkapi.EventInternetUp, func(ctx context.Context) error {
    go func() {
        if err := uploadPendingData(); err != nil {
            api.Logger().Error("upload failed: " + err.Error())
        }
    }()
    return nil
})

events.OnInternetEvent(sdkapi.EventInternetDown, func(ctx context.Context) error {
    api.Logger().Info("machine went offline")
    return nil
})
```

| Constant | When it fires |
|----------|---------------|
| `sdkapi.EventInternetUp` | Machine gains internet access (first boot or after outage) |
| `sdkapi.EventInternetDown` | Internet access lost |

To check the current status without subscribing to events, use [`api.Machine().IsOnline()`](../api/machine-api.md).

---

## Purchase Events

`OnPurchaseEvent` fires when a purchase is confirmed, failed, or cancelled. The callback receives a [`PurchaseEventData`](../api/events-api.md#purchaseeventdata) struct containing the [`IPurchaseRequest`](../api/purchase-request.md) and the associated [`IClientDevice`](../api/client-device.md).

```go
events.OnPurchaseEvent(sdkapi.EventPurchaseSuccess, func(ctx context.Context, data sdkapi.PurchaseEventData) error {
    purchase := data.Purchase
    device := data.Device
    api.Logger().Infof("Purchase %d confirmed for device %s", purchase.ID(), device.MacAddr())
    return nil
})
```

| Constant | When it fires |
|----------|---------------|
| `sdkapi.EventPurchaseSuccess` | Purchase confirmed successfully |
| `sdkapi.EventPurchaseFailed` | Purchase confirmation or execution failed |
| `sdkapi.EventPurchaseCancelled` | User cancelled the purchase |

---

## Voucher Events

There are two callback registrars for vouchers — `OnVoucherEvent` for single-voucher events and `OnVoucherBatchEvent` for batch-level events — because they use distinct Go types (`VoucherEvent` vs `VoucherBatchEvent`) to prevent registering a single-voucher constant with the batch registrar and vice versa.

### Single-Voucher Events

Callbacks receive an [`IVoucher`](../api/voucher-api.md) value.

```go
events.OnVoucherEvent(sdkapi.EventVoucherActivated, func(ctx context.Context, v sdkapi.IVoucher) error {
    api.Logger().Infof("Voucher %s activated", v.Code())
    return nil
})

events.OnVoucherEvent(sdkapi.EventVoucherDeleted, func(ctx context.Context, v sdkapi.IVoucher) error {
    go syncVoucherDeletion(v.UUID())
    return nil
})
```

| Constant | When it fires |
|----------|---------------|
| `sdkapi.EventVoucherBeforeCreate` | **Can cancel** — fires per-voucher inside the batch transaction before INSERT; returning an error rolls back the whole batch |
| `sdkapi.EventVoucherActivated` | Voucher redeemed and session created |
| `sdkapi.EventVoucherUpdated` | Voucher settings updated |
| `sdkapi.EventVoucherDeleted` | Voucher deleted |

#### Cancelling Individual Vouchers (EventVoucherBeforeCreate)

The voucher is an in-memory preview at this point — `ID()` is 0, but `Code()`, `UUID()`, `Type()`, `TimeSecs()`, `DataMb()` and all other param-derived fields are set. Returning an error rolls back the entire batch transaction.

```go
events.OnVoucherEvent(sdkapi.EventVoucherBeforeCreate, func(ctx context.Context, v sdkapi.IVoucher) error {
    if v.TimeSecs() < 300 {
        return fmt.Errorf("voucher time must be at least 5 minutes")
    }
    return nil
})
```

### Batch-Level Voucher Events

Callbacks receive an [`IVoucherBatch`](../api/voucher-api.md) value.

```go
events.OnVoucherBatchEvent(sdkapi.EventVoucherGenerated, func(ctx context.Context, batch sdkapi.IVoucherBatch) error {
    api.Logger().Infof("Batch %s generated %d vouchers", batch.UUID(), batch.VouchersCount())
    return nil
})
```

| Constant | When it fires |
|----------|---------------|
| `sdkapi.EventVoucherBatchBeforeCreate` | **Can cancel** — fires once before any DB writes; returning an error cancels the whole batch |
| `sdkapi.EventVoucherGenerated` | Batch of vouchers successfully created |
| `sdkapi.EventVoucherBatchDeleted` | Voucher batch deleted |

#### Cancelling an Entire Batch (EventVoucherBatchBeforeCreate)

Fires before any INSERTs. The batch preview has `ID() == 0` and `Vouchers()` returns nil, but `UUID()`, `Amount()`, `VouchersCount()`, and `ProviderPkg()` are set. Use this for credit checks, count limits, or quota enforcement.

```go
events.OnVoucherBatchEvent(sdkapi.EventVoucherBatchBeforeCreate, func(ctx context.Context, batch sdkapi.IVoucherBatch) error {
    if batch.VouchersCount() > 100 {
        return fmt.Errorf("cannot create more than 100 vouchers at once")
    }
    return nil
})
```

---

## Complete Example: Multi-Event Plugin

```go
package main

import (
    "context"
    "fmt"

    sdkapi "sdk/api"
)

func main() {}

func Init(api sdkapi.IPluginApi) error {
    events := api.Events()

    // Session sync
    events.OnSessionBatchEvent(sdkapi.EventSessionBatchUpdated, func(ctx context.Context, sessions []sdkapi.IClientSession) error {
        go syncSessions(sessions)
        return nil
    })
    events.OnSessionEvent(sdkapi.EventSessionDeleted, func(ctx context.Context, data sdkapi.SessionEventData) error {
        go syncDeletion(data.Session.UUID())
        return nil
    })

    // Device events
    events.OnClientEvent(sdkapi.EventClientDisconnected, func(ctx context.Context, clnt sdkapi.IClientDevice) error {
        go logDisconnect(clnt.UUID())
        return nil
    })
    events.OnClientMerge(func(ctx context.Context, data sdkapi.EventClientMergeData) error {
        go syncMerge(data.SourceDeviceUUID, data.Target.UUID())
        return nil
    })

    // Voucher sync
    events.OnVoucherBatchEvent(sdkapi.EventVoucherGenerated, func(ctx context.Context, batch sdkapi.IVoucherBatch) error {
        go syncVoucherBatch(batch.UUID())
        return nil
    })
    events.OnVoucherEvent(sdkapi.EventVoucherActivated, func(ctx context.Context, v sdkapi.IVoucher) error {
        go syncVoucherActivation(v.UUID())
        return nil
    })

    // Internet events
    events.OnInternetEvent(sdkapi.EventInternetUp, func(ctx context.Context) error {
        go uploadPending()
        return nil
    })

    return nil
}

func syncSessions(sessions []sdkapi.IClientSession)    { /* ... */ }
func syncDeletion(uuid string)                          { /* ... */ }
func logDisconnect(uuid string)                         { /* ... */ }
func syncMerge(srcUUID, targetUUID string)              { /* ... */ }
func syncVoucherBatch(batchUUID string)                 { /* ... */ }
func syncVoucherActivation(uuid string)                 { /* ... */ }
func uploadPending()                                    { /* ... */ }
```

---

## Best Practices

| Do | Don't |
|----|-------|
| Capture `events := api.Events()` once and reuse | Re-call `api.Events()` inside each registration |
| Spawn a goroutine for any slow callback (I/O, network, DB) | Block the emitter's goroutine with synchronous network calls |
| Return a non-nil error only from `EventClientBeforeConnect`, `EventVoucherBeforeCreate`, or `EventVoucherBatchBeforeCreate` to cancel the operation | Return errors from informational-only events (they are ignored anyway) |
| Register all callbacks during `Init()` | Register callbacks after initialization |

```go
// ❌ Blocks the session timer goroutine for network latency on every session change
events.OnSessionEvent(sdkapi.EventSessionChanged, func(ctx context.Context, data sdkapi.SessionEventData) error {
    return sendToAPI(data.Session) // synchronous HTTP call
})

// ✅ Returns immediately; heavy work runs in background
events.OnSessionEvent(sdkapi.EventSessionChanged, func(ctx context.Context, data sdkapi.SessionEventData) error {
    go sendToAPI(data.Session)
    return nil
})
```

---

## Related

- [IEventsApi](../api/events-api.md) — Complete events API reference with all method signatures, event constants, and supporting types
- [IClientSession](../api/client-session.md) — Session interface (`data.Session` in session callbacks)
- [IClientDevice](../api/client-device.md) — Client device interface (`clnt` in client callbacks)
- [IVouchersApi](../api/voucher-api.md) — Voucher API including `IVoucher` and `IVoucherBatch` interfaces
- [IPurchaseRequest](../api/purchase-request.md) — Purchase request interface (`data.Purchase` in purchase callbacks)
- [IMachineApi](../api/machine-api.md) — Machine API including `IsOnline()` for querying current internet status
