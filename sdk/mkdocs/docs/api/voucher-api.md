# IVouchersApi

The `IVouchersApi` manages voucher lifecycle including creation, activation, and deletion. Each plugin gets its own scoped instance — vouchers are filtered by the plugin's package name.

## Accessing IVouchersApi

```go
vouchersApi := api.Vouchers()
```

---

## IVouchersApi Methods

### Create

Generates a batch of vouchers and returns them. Emits `EventVoucherGenerated` with the created vouchers.

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
    
    vouchers, err := api.Vouchers().Create(r.Context(), params)
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

### List

Returns a paginated list of vouchers for this plugin.

```go
func (w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // List all unactivated vouchers
    isActivated := false
    result, err := api.Vouchers().List(ctx, sdkapi.ListVouchersParams{
        IsActivated: &isActivated,
        Page:        1,
        PerPage:     20,
    })
    if err != nil {
        // handle error
    }
    
    fmt.Printf("Found %d vouchers (total: %d)\n", len(result.Vouchers), result.Count)
    for _, v := range result.Vouchers {
        fmt.Printf("Voucher: %s - %s\n", v.Code(), v.Type())
    }
}
```

**Example: Search vouchers by code or MAC address**

```go
func (w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    searchTerm := "ABC123"
    
    result, err := api.Vouchers().List(ctx, sdkapi.ListVouchersParams{
        Search:  &searchTerm,
        Page:    1,
        PerPage: 20,
    })
    if err != nil {
        // handle error
    }
    
    // Process matching vouchers...
}
```

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

Marks a voucher as used, creates a session based on voucher settings, and associates it with the provided device. Emits `EventVoucherActivated` with the voucher.

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

### GetAvailable

Returns all unactivated vouchers for this plugin.

```go
func (w http.ResponseWriter, r *http.Request) {
    vouchers, err := api.Vouchers().GetAvailable(r.Context())
    if err != nil {
        // handle error
    }
    
    fmt.Printf("Available vouchers: %d\n", len(vouchers))
}
```

### FindVoucherBatch

Retrieves batch metadata by UUID. Returns `nil` if batch not found.

```go
func (w http.ResponseWriter, r *http.Request) {
    batchUUID := "550e8400-e29b-41d4-a716-446655440000"
    
    batch, err := api.Vouchers().FindVoucherBatch(r.Context(), batchUUID)
    if err != nil {
        // handle error
    }
    
    if batch != nil {
        fmt.Printf("Batch created at: %s\n", batch.CreatedAt)
        if batch.TotalAmount != nil {
            fmt.Printf("Total amount: %.2f\n", *batch.TotalAmount)
        }
    }
}
```

### OnVoucherEvent

Registers a callback to be called when a voucher event occurs.

```go
api.Vouchers().OnVoucherEvent(sdkapi.EventVoucherActivated, func(voucher sdkapi.IVoucher) error {
    api.Logger().Info("Voucher %s activated", voucher.Code())
    return nil
})

api.Vouchers().OnVoucherEvent(sdkapi.EventVoucherDeleted, func(voucher sdkapi.IVoucher) error {
    api.Logger().Info("Voucher %s deleted", voucher.Code())
    return nil
})
```

### OnVoucherBatchEvent

Registers a callback to be called when vouchers are generated as a batch.

```go
api.Vouchers().OnVoucherBatchEvent(sdkapi.EventVoucherGenerated, func(vouchers []sdkapi.IVoucher) error {
    api.Logger().Info("Generated %d vouchers", len(vouchers))
    for _, v := range vouchers {
        fmt.Printf("Code: %s\n", v.Code())
    }
    return nil
})
```

### OnBeforeCreate

Registers a hook called before voucher creation. The hook receives a pointer to params and can modify them. Return an error to block creation.

```go
api.Vouchers().OnBeforeCreate(func(ctx context.Context, params *sdkapi.CreateVouchersParams) error {
    // Enforce minimum time
    if params.TimeSecs < 300 {
        params.TimeSecs = 300 // Minimum 5 minutes
    }
    
    // Enforce maximum voucher count
    if params.Count > 100 {
        return fmt.Errorf("cannot create more than 100 vouchers at once")
    }
    
    return nil
})
```

---

## Types

### IVoucher

The `IVoucher` interface represents a single voucher record.

| Method | Return Type | Description |
|--------|-------------|-------------|
| `ID()` | `int64` | Database ID |
| `UUID()` | `string` | Globally unique identifier |
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
| `VoucherExpiresOn()` | `*time.Time` | When the voucher expires (`nil` = never) |
| `ActivatedAt()` | `*time.Time` | When the voucher was activated (`nil` = not activated) |
| `CreatedAt()` | `time.Time` | When the voucher was created |
| `BatchUUID()` | `string` | UUID grouping vouchers created together |

### CreateVouchersParams

Parameters for creating a batch of vouchers:

```go
type CreateVouchersParams struct {
    Count            int        // Number of vouchers to create
    Type             SessionType // "time", "data", or "time-or-data"
    TimeSecs         int64      // Session duration in seconds
    DataMb           int64      // Data allowance in megabytes
    DownSpeedMbps    int64      // Download speed (default 10 Mbps if 0)
    UpSpeedMbps      int64      // Upload speed (default 10 Mbps if 0)
    SessionExpDays   *int       // Days until session expires (nil = never)
    UseGlobal        bool       // Use global bandwidth (default: false)
    VoucherExpiresOn *time.Time // When voucher itself expires (nil = never)
    TotalAmount      *float64   // Optional amount for paid vouchers
    PaymentNote      *string    // Optional payment note
}
```

### UpdateVoucherParams

Parameters for updating a voucher:

```go
type UpdateVoucherParams struct {
    ID               int64      // Voucher database ID
    Type             SessionType
    TimeSecs         int64
    DataMb           int64
    DownSpeedMbps    int64
    UpSpeedMbps      int64
    SessionExpDays   *int       // nil = session never expires
    UseGlobal        bool
    VoucherExpiresOn *time.Time // nil = voucher never expires
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

### ListVouchersParams

Parameters for listing vouchers with pagination:

```go
type ListVouchersParams struct {
    Search      *string // Search by code, provider package, or device MAC
    IsActivated *bool   // Filter by activation status (nil = all)
    Page        int     // Page number (1-indexed)
    PerPage     int     // Results per page
}
```

### ListVouchersResult

Result of listing vouchers:

```go
type ListVouchersResult struct {
    Vouchers []IVoucher // Vouchers for the current page
    Count    int64      // Total count of matching vouchers
}
```

### VoucherBatch

Represents a batch of vouchers with payment metadata:

```go
type VoucherBatch struct {
    ID          int64
    UUID        string
    TotalAmount *float64
    PaymentNote *string
    CreatedAt   time.Time
}
```

### VoucherEvent

The `VoucherEvent` type represents voucher lifecycle events:

```go
type VoucherEvent string

const (
    EventVoucherGenerated    VoucherEvent = "voucher:generated"     // After batch creation
    EventVoucherActivated    VoucherEvent = "voucher:activated"     // When voucher is used
    EventVoucherUpdated      VoucherEvent = "voucher:updated"       // When voucher is modified
    EventVoucherDeleted      VoucherEvent = "voucher:deleted"       // When voucher is removed
    EventVoucherBeforeCreate VoucherEvent = "voucher:before_create" // Before creation hook
)
```

---

## Usage Examples

### Complete Voucher Flow

```go
func Init(api sdkapi.IPluginApi) error {
    // Register voucher event handlers
    api.Vouchers().OnVoucherEvent(sdkapi.EventVoucherActivated, func(v sdkapi.IVoucher) error {
        api.Logger().Info("Voucher %s activated for device %s", 
            v.Code(), v.Device().MacAddr())
        return nil
    })
    
    api.Vouchers().OnVoucherBatchEvent(sdkapi.EventVoucherGenerated, func(vouchers []sdkapi.IVoucher) error {
        api.Logger().Info("Generated %d new vouchers", len(vouchers))
        return nil
    })
    
    return nil
}
```

### Creating Vouchers with Payment Info

```go
func handleCreatePaidVouchers(w http.ResponseWriter, r *http.Request) {
    amount := 100.00
    note := "Bulk purchase - Order #12345"
    expDays := 7
    
    params := sdkapi.CreateVouchersParams{
        Count:          5,
        Type:           sdkapi.SessionTypeTime,
        TimeSecs:       3600, // 1 hour each
        DownSpeedMbps:  10,
        UpSpeedMbps:    5,
        SessionExpDays: &expDays,
        TotalAmount:    &amount,
        PaymentNote:    &note,
    }
    
    vouchers, err := api.Vouchers().Create(r.Context(), params)
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
    if voucher.VoucherExpiresOn() != nil && voucher.VoucherExpiresOn().Before(time.Now()) {
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

## Cloud Sync Integration

The `IVouchersApi` provides event callbacks that enable cloud synchronization of voucher data.

### Syncing Voucher Events to Cloud

```go
func Init(api sdkapi.IPluginApi) error {
    machineID := api.Machine().GetID()
    
    // Sync generated vouchers
    api.Vouchers().OnVoucherBatchEvent(sdkapi.EventVoucherGenerated, func(vouchers []sdkapi.IVoucher) error {
        codes := make([]string, len(vouchers))
        for i, v := range vouchers {
            codes[i] = v.Code()
        }
        return syncToCloud(machineID, "vouchers_generated", map[string]interface{}{
            "batch_uuid": vouchers[0].BatchUUID(),
            "count":      len(vouchers),
            "codes":      codes,
        })
    })
    
    // Sync activations
    api.Vouchers().OnVoucherEvent(sdkapi.EventVoucherActivated, func(v sdkapi.IVoucher) error {
        return syncToCloud(machineID, "voucher_activated", map[string]interface{}{
            "voucher_uuid": v.UUID(),
            "code":         v.Code(),
            "device_uuid":  v.Device().UUID(),
            "session_id":   v.Session().ID(),
        })
    })
    
    return nil
}
```

---

## Related

- [IClientSession](./client-session.md) - Session created when voucher is activated
- [IClientDevice](./client-device.md) - Device associated with activated voucher
- [ISessionsMgrApi](./sessions-mgr-api.md) - Connecting devices after voucher activation
- [IPaymentsApi](./payments-api.md) - Integrating vouchers with payments
