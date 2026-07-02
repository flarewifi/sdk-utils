# IVouchersApi

The `IVouchersApi` manages voucher lifecycle including creation, activation, and deletion. Each plugin gets its own scoped instance — vouchers are filtered by the plugin's package name.

## Accessing IVouchersApi

```go
vouchersApi := api.Vouchers()
```

---

## IVouchersApi Methods

### CreateVouchers

Generates a batch of vouchers and returns them. Emits `EventVoucherBatchCreated` with the created voucher batch.

```go
func (w http.ResponseWriter, r *http.Request) {
    expDays := 30 // Session expires 30 days after activation
    
    params := sdkapi.CreateVouchersParams{
        Count:          10,
        Type:           sdkapi.SessionTypeTimeOrData,
        TimeSecs:       3600,     // 1 hour
        DataMb:         100,      // 100 MB
        DownSpeedMbps:  10,       // 10 Mbps download
        UpSpeedMbps:    5,        // 5 Mbps upload
        SessionExpDays: &expDays, // nil means session never expires
        UseGlobal:      false,    // Use per-user bandwidth
    }
    
    vouchers, err := api.Vouchers().CreateVouchers(r.Context(), params)
    if err != nil {
        // handle error
    }
    
    for _, v := range vouchers {
        fmt.Printf("Created voucher: %s\n", v.Code())
    }
}
```

### FindByCode

Finds an available (unactivated) voucher by its code.

```go
func (w http.ResponseWriter, r *http.Request) {
    code := r.FormValue("voucher_code")
    
    voucher, err := api.Vouchers().FindByCode(r.Context(), code)
    if err != nil {
        // Voucher not found or already activated
        return
    }
    
    fmt.Printf("Found voucher: %s (Type: %s)\n", voucher.Code(), voucher.Type())
}
```

### FindByID

Finds a voucher by its database ID.

```go
func (w http.ResponseWriter, r *http.Request) {
    voucherID := int64(123)
    
    voucher, err := api.Vouchers().FindByID(r.Context(), voucherID)
    if err != nil {
        // Voucher not found
        return
    }
    
    fmt.Printf("Voucher code: %s\n", voucher.Code())
}
```

### Listing & counting vouchers

There is **no voucher list/count method** on `IVouchersApi`. To list, search, paginate,
count, or filter vouchers — including "available" (`activated_at IS NULL`) filtering —
query the core [`vouchers`](../guides/core-database.md#vouchers) table **directly with your
plugin's own sqlc queries**. Your plugin's `sqlc` config already includes the core schema,
so a query against `vouchers` type-checks and generates like any other. See the
[Core Database Tables](../guides/core-database.md) guide for the full schema.

```sql
-- name: SearchVouchers :many
SELECT v.* FROM vouchers v
WHERE v.provider_pkg = @provider_pkg
  AND (@search = '' OR v.code LIKE '%' || @search || '%')
  AND (@only_available = 0 OR v.activated_at IS NULL)
ORDER BY v.created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: CountVouchers :one
SELECT count(*) FROM vouchers WHERE provider_pkg = @provider_pkg;
```

```go
db := queries.New(api.SqlDB())
rows, err := db.SearchVouchers(ctx, queries.SearchVouchersParams{
    ProviderPkg:   api.Info().Package, // scope to vouchers this plugin issued
    Search:        "",
    OnlyAvailable: 1,
    RowLimit:      20,
    RowOffset:     0,
})
```

!!! tip "Scope to your plugin"
    Vouchers carry a `provider_pkg` column. Filter on it (`WHERE provider_pkg = @provider_pkg`,
    passing `api.Info().Package`) to keep a plugin's queries scoped to the vouchers it
    issued — the removed SDK methods did this automatically.

### Update

Changes a voucher's session type, time, data, and speed settings. Emits `EventVoucherUpdated` with the updated voucher.

```go
func (w http.ResponseWriter, r *http.Request) {
    expDays := 60
    
    params := sdkapi.UpdateVoucherParams{
        ID:             123,
        Type:           sdkapi.SessionTypeTime,
        TimeSecs:       7200,     // 2 hours
        DataMb:         0,        // Not applicable for time-only
        DownSpeedMbps:  20,
        UpSpeedMbps:    10,
        SessionExpDays: &expDays,
        UseGlobal:      false,
    }
    
    voucher, err := api.Vouchers().Update(r.Context(), params)
    if err != nil {
        // handle error
    }
    
    fmt.Printf("Updated voucher: %s\n", voucher.Code())
}
```

### Activate

Marks a voucher as used, creates a session based on voucher settings, and associates it with the provided device. First emits `EventVoucherBeforeActivate` — a subscriber returning an error cancels activation before any session is created — then emits `EventVoucherActivated` with the voucher after success.

```go
func (w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Get the client device from the request
    device, err := api.Http().GetClientDevice(r)
    if err != nil {
        // handle error
        return
    }
    
    // Find the voucher by code
    code := r.FormValue("voucher_code")
    voucher, err := api.Vouchers().FindByCode(ctx, code)
    if err != nil {
        // Invalid voucher code
        return
    }
    
    // Activate the voucher
    result, err := api.Vouchers().Activate(ctx, sdkapi.ActivateVoucherParams{
        ID:     voucher.ID(),
        Device: device,
    })
    if err != nil {
        // Activation failed
        return
    }
    
    fmt.Printf("Activated voucher: %s\n", result.Voucher.Code())
    fmt.Printf("Created session ID: %d\n", result.Session.ID())
    
    // Connect the device to the internet
    api.SessionsMgr().Connect(ctx, device, "You are now connected!")
}
```

### Delete

Removes a voucher by its ID. Emits `EventVoucherDeleted` with the deleted voucher.

```go
func (w http.ResponseWriter, r *http.Request) {
    voucherID := int64(123)
    
    err := api.Vouchers().Delete(r.Context(), voucherID)
    if err != nil {
        // handle error
    }
}
```

### DeleteActivated

Removes all activated vouchers for this plugin. Emits `EventVoucherDeleted` for each deleted voucher.

```go
func (w http.ResponseWriter, r *http.Request) {
    err := api.Vouchers().DeleteActivated(r.Context())
    if err != nil {
        // handle error
    }
}
```

!!! note "Available vouchers & per-batch counts"
    `GetAvailable` and `GetVouchersByBatchUUIDCount` have been removed. Query the core
    `vouchers` table directly instead — filter `activated_at IS NULL` for available
    vouchers, or `batch_uuid = @batch_uuid` to count/list a batch's vouchers (see
    [Listing & counting vouchers](#listing-counting-vouchers) above). Inside an
    `EventVoucherBatchBeforeCreate` hook the rows don't exist yet, so use the batch
    object's [`VouchersCount()`](#ivoucherbatch) for the intended count.

### FindBatchByUUID

Finds a voucher batch by its UUID.

```go
func (w http.ResponseWriter, r *http.Request) {
    batchUUID := "550e8400-e29b-41d4-a716-446655440000"
    
    batch, err := api.Vouchers().FindBatchByUUID(r.Context(), batchUUID)
    if err != nil {
        // handle error
    }
    
    fmt.Printf("Batch created at: %s\n", batch.CreatedAt())
    if batch.Amount() != nil {
        fmt.Printf("Total amount: %.2f\n", *batch.Amount())
    }
}
```

### FindBatchByCode

Finds a batch that contains a voucher with the given code.

```go
func (w http.ResponseWriter, r *http.Request) {
    code := "ABC123XYZ"
    
    batch, err := api.Vouchers().FindBatchByCode(r.Context(), code)
    if err != nil {
        // handle error
    }
    
    fmt.Printf("Batch UUID: %s\n", batch.UUID())
    fmt.Printf("Vouchers in batch: %d\n", batch.VouchersCount())
}
```

### UpdateBatch

Updates a voucher batch's amount and metadata.

```go
func (w http.ResponseWriter, r *http.Request) {
    amount := 150.00
    metadata := `{"order_id": "12345", "customer": "John Doe"}`
    
    batch, err := api.Vouchers().UpdateBatch(r.Context(), sdkapi.UpdateVoucherBatchParams{
        UUID:     "550e8400-e29b-41d4-a716-446655440000",
        Amount:   &amount,
        Metadata: metadata,
    })
    if err != nil {
        // handle error
    }
    
    fmt.Printf("Updated batch: %s\n", batch.UUID())
}
```

### Listing & counting batches

There is **no batch list/count method** on `IVouchersApi`. To list, search, paginate, or
count voucher batches, query the core
[`voucher_batches`](../guides/core-database.md#voucher_batches) table **directly with your
plugin's own sqlc queries** (the core schema is already available to your plugin's sqlc):

```sql
-- name: SearchVoucherBatches :many
SELECT * FROM voucher_batches
WHERE provider_pkg = @provider_pkg
ORDER BY created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: CountVoucherBatches :one
SELECT count(*) FROM voucher_batches WHERE provider_pkg = @provider_pkg;
```

Use [`FindBatchByUUID`](#findbatchbyuuid) / [`FindBatchByCode`](#findbatchbycode) when you
need a single batch wrapped as an `IVoucherBatch`.

### DeleteBatch

Removes a voucher batch and all its vouchers by UUID. First emits `EventVoucherBatchBeforeDelete` — a subscriber returning an error cancels the deletion before any row is removed — then emits `EventVoucherBatchDeleted` with the deleted batch after success.

```go
func (w http.ResponseWriter, r *http.Request) {
    batchUUID := "550e8400-e29b-41d4-a716-446655440000"
    
    err := api.Vouchers().DeleteBatch(r.Context(), batchUUID)
    if err != nil {
        // handle error
    }
    
    fmt.Printf("Deleted batch: %s\n", batchUUID)
}
```

### OnVoucherEvent

!!! warning "Deprecated"
    Use `api.Events().OnVoucherEvent(...)` instead.

Registers a callback to be called when a voucher event occurs.

```go
api.Vouchers().OnVoucherEvent(sdkapi.EventVoucherActivated, func(ctx context.Context, v sdkapi.IVoucher) error {
    api.Logger().Info("Voucher %s activated", v.Code())
    return nil
})

api.Vouchers().OnVoucherEvent(sdkapi.EventVoucherDeleted, func(ctx context.Context, v sdkapi.IVoucher) error {
    api.Logger().Info("Voucher %s deleted", v.Code())
    return nil
})
```

### OnVoucherBatchEvent

!!! warning "Deprecated"
    Use `api.Events().OnVoucherBatchEvent(...)` instead.

Registers a callback to be called when voucher batch events occur.

```go
api.Vouchers().OnVoucherBatchEvent(sdkapi.EventVoucherBatchCreated, func(ctx context.Context, batch sdkapi.IVoucherBatch) error {
    api.Logger().Info("Generated batch %s with %d vouchers", batch.UUID(), batch.VouchersCount())
    return nil
})
```

### Before-Create Hooks

Use `api.Events()` to intercept voucher creation before any DB writes:

- **Batch-level check** (fires once, before any DB writes): `OnVoucherBatchEvent(sdkapi.EventVoucherBeforeCreate, ...)`
- **Per-voucher check** (fires inside the transaction, before each INSERT): `OnVoucherEvent(sdkapi.EventVoucherBeforeCreate, ...)`

```go
// Batch-level: enforce count or credit limits
api.Events().OnVoucherBatchEvent(sdkapi.EventVoucherBeforeCreate, func(ctx context.Context, batch sdkapi.IVoucherBatch) error {
    if batch.VouchersCount() > 100 {
        return fmt.Errorf("cannot create more than 100 vouchers at once")
    }
    return nil
})

// Per-voucher: inspect individual voucher settings
api.Events().OnVoucherEvent(sdkapi.EventVoucherBeforeCreate, func(ctx context.Context, v sdkapi.IVoucher) error {
    if v.TimeSecs() < 300 {
        return fmt.Errorf("voucher time must be at least 5 minutes")
    }
    return nil
})
```

See [IEventsApi](./events-api.md#eventvoucherbeforecreate-batch-level-can-cancel) for full documentation.

---

## Types

### IVoucher

The `IVoucher` interface represents a single voucher record.

| Method | Return Type | Description |
|--------|-------------|-------------|
| `ID()` | `int64` | Database ID |
| `UUID()` | `string` | Globally unique identifier |
| `BatchUUID()` | `string` | UUID grouping vouchers created together |
| `Code()` | `string` | Voucher code for activation |
| `ProviderPkg()` | `string` | Plugin package that generated the voucher |
| `Type()` | `SessionType` | Session type: `"time"`, `"data"`, or `"time-or-data"` |
| `TimeSecs()` | `int64` | Session duration in seconds |
| `DataMb()` | `int64` | Data allowance in megabytes |
| `DownSpeedMbps()` | `int64` | Download speed in Mbps |
| `UpSpeedMbps()` | `int64` | Upload speed in Mbps |
| `SessionExpDays()` | `*int` | Days until session expires after activation (`nil` = never) |
| `UseGlobal()` | `bool` | `true` = use global bandwidth, `false` = per-user |
| `Session()` | `IClientSession` | Associated session (only for activated vouchers) |
| `Device()` | `IClientDevice` | Associated device (only for activated vouchers) |
| `ExpiresAt()` | `*time.Time` | When the voucher expires (`nil` = never) |
| `ActivatedAt()` | `*time.Time` | When the voucher was activated (`nil` = not activated) |
| `CreatedAt()` | `time.Time` | When the voucher was created |

### IVoucherBatch

The `IVoucherBatch` interface represents a batch of vouchers with metadata.

| Method | Return Type | Description |
|--------|-------------|-------------|
| `ID()` | `int64` | Database ID |
| `UUID()` | `string` | Globally unique identifier for the batch |
| `Amount()` | `*float64` | Optional amount associated with the batch (e.g., paid vouchers) |
| `Metadata()` | `string` | JSON metadata string for custom data |
| `ProviderPkg()` | `string` | Plugin package that created this batch |
| `VouchersCount()` | `int64` | Number of vouchers in the batch. On a persisted batch this counts the rows; inside an `EventVoucherBatchBeforeCreate` hook the batch is a preview and this is the *intended* count before any DB write |
| `CreatedAt()` | `time.Time` | When the batch was created |
| `UpdatedAt()` | `time.Time` | When the batch was last updated |

!!! note "Listing a batch's vouchers"
    `IVoucherBatch` no longer has a `Vouchers()` method. To list the vouchers in a batch,
    query the core `vouchers` table with `WHERE batch_uuid = <UUID()>` via your plugin's
    own sqlc query — see [Listing & counting vouchers](#listing-counting-vouchers).

### CreateVouchersParams

Parameters for creating a batch of vouchers:

```go
type CreateVouchersParams struct {
    Count          int        // Number of vouchers to create
    Type           SessionType // "time", "data", or "time-or-data"
    TimeSecs       int64      // Session duration in seconds
    DataMb         int64      // Data allowance in megabytes
    DownSpeedMbps  int64      // Download speed (default 10 Mbps if 0)
    UpSpeedMbps    int64      // Upload speed (default 10 Mbps if 0)
    SessionExpDays *int       // Days until session expires (nil = never)
    UseGlobal      bool       // Use global bandwidth (default: false)
    ExpiresAt      *time.Time // When voucher itself expires (nil = never)
    BatchUUID      string     // Optional - if empty, a UUID will be generated
    Amount         *float64   // Optional amount for the voucher batch
}
```

### UpdateVoucherParams

Parameters for updating a voucher:

```go
type UpdateVoucherParams struct {
    ID             int64      // Voucher database ID
    Type           SessionType
    TimeSecs       int64
    DataMb         int64
    DownSpeedMbps  int64
    UpSpeedMbps    int64
    SessionExpDays *int       // nil = session never expires
    UseGlobal      bool
    ExpiresAt      *time.Time // nil = voucher never expires
}
```

### ActivateVoucherParams

Parameters for activating a voucher:

```go
type ActivateVoucherParams struct {
    ID     int64         // Voucher ID
    Device IClientDevice // Device to associate with the voucher and session
}
```

### VoucherActivateResult

Result of activating a voucher:

```go
type VoucherActivateResult struct {
    Voucher IVoucher       // The activated voucher
    Session IClientSession // The session created from the voucher
}
```

### UpdateVoucherBatchParams

Parameters for updating a voucher batch:

```go
type UpdateVoucherBatchParams struct {
    UUID     string   // Batch UUID to update
    Amount   *float64 // New amount (nil to clear)
    Metadata string   // New metadata (empty string to clear)
}
```

### VoucherEvent

The `VoucherEvent` type represents voucher lifecycle events:

```go
type VoucherEvent string

// Single-voucher events (use with OnVoucherEvent)
const (
    EventVoucherBeforeCreate   VoucherEvent = "voucher:before_create"   // Per-voucher pre-create (cancellable)
    EventVoucherBeforeActivate VoucherEvent = "voucher:before_activate" // Pre-activate (cancellable)
    EventVoucherActivated      VoucherEvent = "voucher:activated"       // When voucher is activated
    EventVoucherUpdated        VoucherEvent = "voucher:updated"         // When voucher is modified
    EventVoucherDeleted        VoucherEvent = "voucher:deleted"         // When voucher is removed
)

// Voucher-batch events (use with OnVoucherBatchEvent)
const (
    EventVoucherBatchBeforeCreate VoucherBatchEvent = "voucher:before_create"        // Batch pre-create (cancellable)
    EventVoucherBatchCreated      VoucherBatchEvent = "voucher:batch_created"         // After successful batch creation (was EventVoucherGenerated)
    EventVoucherBatchBeforeDelete VoucherBatchEvent = "voucher:batch_before_delete"  // Batch pre-delete (cancellable)
    EventVoucherBatchDeleted      VoucherBatchEvent = "voucher:batch_deleted"         // When batch is deleted
)
```

---

## Usage Examples

### Complete Voucher Flow

```go
func Init(api sdkapi.IPluginApi) error {
    // Register voucher event handlers
    api.Vouchers().OnVoucherEvent(sdkapi.EventVoucherActivated, func(ctx context.Context, v sdkapi.IVoucher) error {
        api.Logger().Info("Voucher %s activated for device %s",
            v.Code(), v.Device().MacAddr())
        return nil
    })

    api.Vouchers().OnVoucherBatchEvent(sdkapi.EventVoucherBatchCreated, func(ctx context.Context, batch sdkapi.IVoucherBatch) error {
        api.Logger().Info("Generated batch %s with %d new vouchers", batch.UUID(), batch.VouchersCount())
        return nil
    })

    return nil
}
```

### Creating Vouchers with Batch Metadata

```go
func handleCreateVouchers(w http.ResponseWriter, r *http.Request) {
    amount := 100.00
    expDays := 7
    
    params := sdkapi.CreateVouchersParams{
        Count:          5,
        Type:           sdkapi.SessionTypeTime,
        TimeSecs:       3600, // 1 hour each
        DownSpeedMbps:  10,
        UpSpeedMbps:    5,
        SessionExpDays: &expDays,
        Amount:         &amount,
    }
    
    vouchers, err := api.Vouchers().CreateVouchers(r.Context(), params)
    if err != nil {
        // handle error
        return
    }
    
    // Return voucher codes to customer
    for _, v := range vouchers {
        fmt.Printf("Voucher: %s (Batch: %s)\n", v.Code(), v.BatchUUID())
    }
}
```

### Voucher Activation with Validation

```go
func handleActivateVoucher(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    code := r.FormValue("code")
    
    // Get client device
    device, err := api.Http().GetClientDevice(r)
    if err != nil {
        api.Http().Response().Json(w, r, map[string]string{
            "error": "Device not found",
        }, http.StatusBadRequest)
        return
    }
    
    // Find voucher
    voucher, err := api.Vouchers().FindByCode(ctx, code)
    if err != nil {
        api.Http().Response().Json(w, r, map[string]string{
            "error": "Invalid voucher code",
        }, http.StatusNotFound)
        return
    }
    
    // Check if voucher has expired
    if voucher.ExpiresAt() != nil && voucher.ExpiresAt().Before(time.Now()) {
        api.Http().Response().Json(w, r, map[string]string{
            "error": "Voucher has expired",
        }, http.StatusBadRequest)
        return
    }
    
    // Activate voucher
    result, err := api.Vouchers().Activate(ctx, sdkapi.ActivateVoucherParams{
        ID:     voucher.ID(),
        Device: device,
    })
    if err != nil {
        api.Http().Response().Json(w, r, map[string]string{
            "error": "Activation failed",
        }, http.StatusInternalServerError)
        return
    }
    
    // Connect device
    api.SessionsMgr().Connect(ctx, device, "Connected with voucher!")
    
    api.Http().Response().Json(w, r, map[string]interface{}{
        "success":    true,
        "session_id": result.Session.ID(),
        "time_secs":  result.Session.TimeSecs(),
        "data_mb":    result.Session.DataMb(),
    }, http.StatusOK)
}
```

---

## Related

- [IClientSession](./client-session.md) - Session created when voucher is activated
- [IClientDevice](./client-device.md) - Device associated with activated voucher
- [ISessionsMgrApi](./sessions-mgr-api.md) - Connecting devices after voucher activation
- [IPaymentsApi](./payments-api.md) - Integrating vouchers with payments
