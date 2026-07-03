# IPurchaseRequest

The `IPurchaseRequest` interface represents a purchase request record in the Flarewifi payment system. It provides methods to manage the lifecycle of a purchase, including creating payments and confirming or canceling purchases.

Purchase requests are created through [IPaymentsApi.Checkout()](./payments-api.md#checkout) and can be retrieved using `api.Payments().GetPurchaseRequest(r)` or `api.Payments().FindPurchaseRequestByUUID(uuid)`.

---

## IPurchaseRequest Methods

The following methods are available in `IPurchaseRequest`:

### ID

Returns the database ID of the purchase request.

```go
id := purchaseRequest.ID()
fmt.Printf("Purchase ID: %d\n", id)
```

### UUID

Returns the unique identifier (UUID) of the purchase request.

```go
uuid := purchaseRequest.UUID()
fmt.Printf("Purchase UUID: %s\n", uuid)
```

### DeviceID

Returns the device ID associated with the purchase.

```go
deviceID := purchaseRequest.DeviceID()
fmt.Printf("Device ID: %d\n", deviceID)
```

### Sku

Returns the SKU (Stock Keeping Unit) of the purchase item.

```go
sku := purchaseRequest.Sku()
fmt.Printf("SKU: %s\n", sku)
```

### Name

Returns the name of the purchase item.

```go
name := purchaseRequest.Name()
fmt.Printf("Item: %s\n", name)
```

### Description

Returns the description of the purchase item.

```go
description := purchaseRequest.Description()
fmt.Printf("Description: %s\n", description)
```

### Price

Returns the price of the purchase item.

```go
price := purchaseRequest.Price()
fmt.Printf("Price: $%.2f\n", price)
```

### AnyPrice

Returns `true` if the purchase request allows any price (flexible/donation-style pricing).

```go
if purchaseRequest.AnyPrice() {
    fmt.Println("Flexible price purchase")
}
```

### IsFixedPrice

Returns `true` if the purchase request has a fixed price.

```go
if purchaseRequest.IsFixedPrice() {
    fmt.Println("Fixed price purchase")
} else {
    fmt.Println("Flexible price purchase")
}
```

### ConfirmedAt

Returns the timestamp when the purchase was confirmed, or `nil` if not yet confirmed.

```go
confirmedAt := purchaseRequest.ConfirmedAt()
if confirmedAt != nil {
    fmt.Printf("Confirmed at: %s\n", confirmedAt.Format(time.RFC3339))
}
```

### CancelledAt

Returns the timestamp when the purchase was cancelled, or `nil` if not cancelled.

```go
cancelledAt := purchaseRequest.CancelledAt()
if cancelledAt != nil {
    fmt.Printf("Cancelled at: %s\n", cancelledAt.Format(time.RFC3339))
}
```

### CancelledReason

Returns the reason for cancellation if cancelled, or `nil` if not cancelled.

```go
reason := purchaseRequest.CancelledReason()
if reason != nil {
    fmt.Printf("Cancellation reason: %s\n", *reason)
}
```

### CreatedAt

Returns the timestamp when the purchase was created.

```go
createdAt := purchaseRequest.CreatedAt()
fmt.Printf("Created at: %s\n", createdAt.Format(time.RFC3339))
```

### CallbackPluginPkg

Returns the callback plugin package name.

```go
pkg := purchaseRequest.CallbackPluginPkg()
fmt.Printf("Callback plugin: %s\n", pkg)
```

### CallbackRoute

Returns the callback route name for the purchase.

```go
route := purchaseRequest.CallbackRoute()
fmt.Printf("Callback route: %s\n", route)
```

### Metadata

Returns the metadata map associated with the purchase.

```go
metadata := purchaseRequest.Metadata()
for key, value := range metadata {
    fmt.Printf("%s: %s\n", key, value)
}
```

### IsConfirmed

Returns `true` if the purchase has been confirmed.

```go
if purchaseRequest.IsConfirmed() {
    fmt.Println("Purchase is confirmed")
}
```

### IsCancelled

Returns `true` if the purchase has been cancelled.

```go
if purchaseRequest.IsCancelled() {
    fmt.Println("Purchase is cancelled")
}
```

### Processing

Returns `true` if the purchase is currently being processed.

```go
if purchaseRequest.Processing() {
    fmt.Println("Payment is being processed")
}
```

### PaymentUrl

Returns the external payment URL for the purchase (if any).

```go
url := purchaseRequest.PaymentUrl()
if url != "" {
    fmt.Printf("Payment URL: %s\n", url)
}
```

### SetProcessing

Sets the processing state and payment URL for the purchase. If `paymentUrl` is empty, it clears the processing state. If `paymentUrl` is provided, it sets processing to `true` and stores the URL.

```go
ctx := r.Context()

// Set processing with external payment URL
err := purchaseRequest.SetProcessing(ctx, "https://payment-gateway.com/pay/abc123")
if err != nil {
    // handle error
}

// Clear processing state
err = purchaseRequest.SetProcessing(ctx, "")
```

### CreatePayment

Creates a payment record for the purchase request.

```go
ctx := r.Context()

params := sdkapi.CreatePaymentParams{
    Amount:        29.99,
    PaymentMethod: "Credit Card", // optional, label shown in the sales inventory
}

err := purchaseRequest.CreatePayment(ctx, params)
if err != nil {
    // handle error
}
```

### GetPaymentData

Returns the total accumulated payment and the first payment's provider/method.

```go
ctx := r.Context()

payment, err := purchaseRequest.GetPaymentData(ctx)
if err != nil {
    // handle error
}

fmt.Printf("Purchase ID: %d\n", payment.PurchaseID)
fmt.Printf("Total Payment: $%.2f\n", payment.TotalPayment)
fmt.Printf("Payment Provider: %s\n", payment.PaymentProvider)
fmt.Printf("Payment Method: %s\n", payment.PaymentMethod)
```

### Execute

Executes the purchase by invoking the callback plugin's registered `PurchaseExecuteHandler` (see `IPaymentsApi.HandlePurchaseExecute`) in-process. No HTTP request is made. The params carry the success status and amount passed to that handler.

```go
ctx := r.Context()

params := sdkapi.ExecuteParams{
    Amount:  29.99,
    Success: true,
    Message: "Payment successful",
}

err := purchaseRequest.Execute(ctx, params)
if err != nil {
    // handle error
}
```

### RedirectToCallback

Redirects the user to the callback route of the purchase request.

```go
purchaseRequest.RedirectToCallback(w, r)
// User is redirected - this should be the last line in the handler
```

### Confirm

Confirms the purchase. This must be called in the purchase execute handler after successful payment.

```go
ctx := r.Context()

err := purchaseRequest.Confirm(ctx)
if err != nil {
    // handle error
}
// Purchase is now confirmed
```

### Cancel

Cancels the purchase. This should be called in the purchase execute handler when payment fails.

```go
ctx := r.Context()

err := purchaseRequest.Cancel(ctx)
if err != nil {
    // handle error
}
// Purchase is now cancelled
```

### UpdateMetadata

Updates the metadata associated with the purchase. This should be called before `Confirm()` to ensure metadata is available for sync.

```go
ctx := r.Context()

metadata := map[string]string{
    "transaction_id": "txn_12345",
    "payment_method": "credit_card",
    "receipt_number": "RCP-2024-001",
}

err := purchaseRequest.UpdateMetadata(ctx, metadata)
if err != nil {
    // handle error
}

// Now confirm the purchase
err = purchaseRequest.Confirm(ctx)
```

---

## Types

### PurchaseRequest

The `PurchaseRequest` struct represents the initial purchase request data passed to `Checkout()`:

```go
type PurchaseRequest struct {
    Sku           string            // Stock keeping unit identifier
    Name          string            // Display name of the item
    Description   string            // Description of the item
    Price         float64           // Price of the item
    AnyPrice      bool              // Whether flexible pricing is allowed
    CallbackRoute string            // Route name to redirect after payment
    Metadata      map[string]string // Additional metadata
    Processing    bool              // Whether payment is being processed
    PaymentUrl    string            // External payment URL (if any)
}
```

### PurchasePaymentData

The `PurchasePaymentData` struct is returned by `GetPaymentData()`:

```go
type PurchasePaymentData struct {
    PurchaseID      int64   `json:"purchase_id"`      // Database ID of the purchase
    TotalPayment    float64 `json:"total_payment"`    // Total amount paid
    PaymentProvider string  `json:"payment_provider"` // Calling plugin's package name
    PaymentMethod   string  `json:"payment_method"`   // Method label; see below
}
```

`PaymentMethod` is the method label passed to `CreatePayment` (e.g. "Coins",
"Credit Card") if one was given. If it was left blank, it falls back to the
`PaymentProvider` plugin's `plugin.json` `"name"` field (e.g. "Wired
Coinslot"), or to the raw `PaymentProvider` package string if that plugin
isn't currently installed. It is only blank when `PaymentProvider` itself is
blank (no payment recorded yet).

### CreatePaymentParams

The `CreatePaymentParams` struct is used when creating a payment:

```go
type CreatePaymentParams struct {
    Amount        float64 // Payment amount
    PaymentMethod string  // Optional method label (e.g. "Coins", "Credit Card")
}
```

The payment's `provider` (which plugin processed it) is derived automatically
from the calling plugin's package name — it is not part of `CreatePaymentParams`.

### ExecuteParams

The `ExecuteParams` struct holds parameters for executing a purchase:

```go
type ExecuteParams struct {
    Amount  float64 `json:"amount"`  // Payment amount
    Success bool    `json:"success"` // Whether payment was successful
    Message string  `json:"message"` // Status message
}
```

---

## Purchase Lifecycle

1. **Create**: Purchase request is created via `api.Payments().Checkout()`
2. **Payment Selection**: User selects a payment option from available providers
3. **Payment Processing**: Payment is processed via `CreatePayment()`
4. **Execute**: Payment provider calls `Execute()` with result
5. **Confirmation/Cancellation**: Execute handler calls `Confirm()` or `Cancel()`
6. **Callback Redirect**: User is redirected via `RedirectToCallback()`

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Checkout   │────▶│  Payment    │────▶│  Execute    │
│  (Create)   │     │  Selection  │     │  (in-proc)  │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                    ┌─────────────┐     ┌──────▼──────┐
                    │  Callback   │◀────│  Confirm/   │
                    │  Redirect   │     │  Cancel     │
                    └─────────────┘     └─────────────┘
```

For a detailed payment flow diagram and implementation guide, see [Accept Payments](../tutorials/accepting-payments.md).

---

## Usage Examples

### Payment Provider Handler

```go
// Handler for a payment option (e.g., credit card)
func handleCreditCardPayment(w http.ResponseWriter, r *http.Request) {
    // Get the pending purchase request
    purchaseReq, err := api.Payments().GetPurchaseRequest(r)
    if err != nil {
        api.Http().Response().FlashMsg(w, r, "No pending purchase", sdkapi.FlashMsgError)
        return
    }

    ctx := r.Context()

    // Create payment record. Provider is derived automatically from this
    // plugin's package name; PaymentMethod is an optional display label.
    params := sdkapi.CreatePaymentParams{
        Amount:        purchaseReq.Price(),
        PaymentMethod: "Credit Card",
    }

    err = purchaseReq.CreatePayment(ctx, params)
    if err != nil {
        api.Http().Response().FlashMsg(w, r, "Failed to create payment", sdkapi.FlashMsgError)
        return
    }

    // Set external payment URL and redirect user
    externalUrl := "https://payment-gateway.com/pay/" + purchaseReq.UUID()
    err = purchaseReq.SetProcessing(ctx, externalUrl)
    if err != nil {
        // handle error
        return
    }

    http.Redirect(w, r, externalUrl, http.StatusFound)
}
```

### Execute Handler

Registered once in `Init` via `HandlePurchaseExecute`. The core invokes it
in-process when a provider calls `Execute()` — there is no HTTP request to parse;
the payment result arrives as `ExecuteParams`.

```go
func Init(api sdkapi.IPluginApi) error {
    api.Payments().HandlePurchaseExecute(func(ctx context.Context, purchaseReq sdkapi.IPurchaseRequest, params sdkapi.ExecuteParams) error {
        if params.Success {
            // Optionally update metadata before confirming
            if err := purchaseReq.UpdateMetadata(ctx, map[string]string{
                "transaction_id": "txn_from_gateway",
            }); err != nil {
                api.Logger().Error("Failed to update metadata: %v", err)
            }

            // Confirm the purchase
            if err := purchaseReq.Confirm(ctx); err != nil {
                api.Logger().Error("Failed to confirm purchase: %v", err)
                return err
            }

            // Execute your business logic (branch on purchaseReq.Sku() if needed)
            grantAccess(purchaseReq.DeviceID(), purchaseReq.Metadata())
        } else {
            // Cancel the purchase
            if err := purchaseReq.Cancel(ctx); err != nil {
                api.Logger().Error("Failed to cancel purchase: %v", err)
            }
        }
        return nil
    })
    return nil
}
```

### Callback Handler

```go
func handlePaymentCallback(w http.ResponseWriter, r *http.Request) {
    purchaseReq, err := api.Payments().GetPurchaseRequest(r)
    if err != nil {
        // No purchase found, redirect to home
        http.Redirect(w, r, "/", http.StatusFound)
        return
    }

    if purchaseReq.IsConfirmed() {
        // Payment successful
        api.Http().Response().FlashMsg(w, r, "Payment successful!", sdkapi.FlashMsgSuccess)
        purchaseReq.RedirectToCallback(w, r)
    } else if purchaseReq.IsCancelled() {
        // Payment cancelled
        api.Http().Response().FlashMsg(w, r, "Payment was cancelled", sdkapi.FlashMsgWarning)
        http.Redirect(w, r, "/", http.StatusFound)
    } else if purchaseReq.Processing() {
        // Still processing - show waiting page
        api.Http().Response().Render(w, r, views.ProcessingPayment(purchaseReq), nil)
    } else {
        // Unknown state
        http.Redirect(w, r, "/", http.StatusFound)
    }
}
```

---

## Related

- [IPaymentsApi](./payments-api.md) - Payment API for checkout and providers
- [IHttpRouterApi](./http-router-api.md) - Setting up routes for payment handlers
