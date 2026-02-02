# IPurchaseRequest

The `IPurchaseRequest` interface represents a purchase request record in the Flare Hotspot payment system. It provides methods to manage the lifecycle of a purchase, including creating payments, handling wallet payments, and confirming or canceling purchases.

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

### WalletDebit

Returns the wallet debit amount for the purchase.

```go
walletDebit := purchaseRequest.WalletDebit()
fmt.Printf("Wallet debit: $%.2f\n", walletDebit)
```

### WalletTxID

Returns the wallet transaction ID if available, or `nil` if no wallet payment was made.

```go
txID := purchaseRequest.WalletTxID()
if txID != nil {
    fmt.Printf("Wallet Transaction ID: %d\n", *txID)
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

### WebHookRoute

Returns the webhook route name for the purchase.

```go
route := purchaseRequest.WebHookRoute()
fmt.Printf("Webhook route: %s\n", route)
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
    Amount:  29.99,
    Optname: "credit_card",
}

err := purchaseRequest.CreatePayment(ctx, params)
if err != nil {
    // handle error
}
```

### PayWithWallet

Pays for the purchase using the customer's wallet balance. The amount will be debited from the wallet once the purchase request has been confirmed.

```go
ctx := r.Context()

amount := 10.00
err := purchaseRequest.PayWithWallet(ctx, amount)
if err != nil {
    // handle insufficient funds or other error
}
```

### State

Returns the current state of the purchase, including payment totals and wallet information.

```go
ctx := r.Context()

state, err := purchaseRequest.State(ctx)
if err != nil {
    // handle error
}

fmt.Printf("Purchase ID: %d\n", state.PurchaseID)
fmt.Printf("Total Payment: $%.2f\n", state.TotalPayment)
fmt.Printf("Wallet Debit: $%.2f\n", state.WalletDebit)
fmt.Printf("Wallet Ending Balance: $%.2f\n", state.WalletEndingBal)
fmt.Printf("Wallet Real Balance: $%.2f\n", state.WalletRealBal)
```

### Execute

Executes the webhook for the purchase. This makes an internal POST request to the webhook route. The params contain the success status and message to be passed to the webhook handler.

```go
ctx := r.Context()

params := sdkapi.ExecuteParams{
    DeviceID:    purchaseRequest.DeviceID(),
    PurchaseUID: purchaseRequest.UUID(),
    Amount:      29.99,
    Success:     true,
    Message:     "Payment successful",
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

Confirms the purchase. This must be called in the purchase webhook handler after successful payment.

```go
ctx := r.Context()

err := purchaseRequest.Confirm(ctx)
if err != nil {
    // handle error
}
// Purchase is now confirmed
```

### Cancel

Cancels the purchase. This should be called in the purchase webhook handler when payment fails.

```go
ctx := r.Context()

err := purchaseRequest.Cancel(ctx)
if err != nil {
    // handle error
}
// Purchase is now cancelled
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
    WebHookRoute  string            // Route name for payment webhook
    Metadata      map[string]string // Additional metadata
    Processing    bool              // Whether payment is being processed
    PaymentUrl    string            // External payment URL (if any)
}
```

### PurchaseState

The `PurchaseState` struct represents the current state of a purchase:

```go
type PurchaseState struct {
    PurchaseID      int64   `json:"purchase_id"`       // Database ID of the purchase
    TotalPayment    float64 `json:"total_payment"`     // Total amount paid
    WalletDebit     float64 `json:"wallet_debit"`      // Amount debited from wallet
    WalletEndingBal float64 `json:"wallet_ending_bal"` // Wallet balance after purchase
    WalletRealBal   float64 `json:"wallet_real_bal"`   // Actual wallet balance
}
```

### CreatePaymentParams

The `CreatePaymentParams` struct is used when creating a payment:

```go
type CreatePaymentParams struct {
    Amount  float64 // Payment amount
    Optname string  // Payment option name (e.g., "credit_card", "gcash")
}
```

### ExecuteParams

The `ExecuteParams` struct holds parameters for executing a purchase webhook:

```go
type ExecuteParams struct {
    DeviceID    int64   `json:"device_id"`    // Device ID making the purchase
    PurchaseUID string  `json:"purchase_uid"` // UUID of the purchase request
    Amount      float64 `json:"amount"`       // Payment amount
    Success     bool    `json:"success"`      // Whether payment was successful
    Message     string  `json:"message"`      // Status message
}
```

---

## Purchase Lifecycle

1. **Create**: Purchase request is created via `api.Payments().Checkout()`
2. **Payment Selection**: User selects a payment option from available providers
3. **Payment Processing**: Payment is processed via `CreatePayment()` or `PayWithWallet()`
4. **Webhook Execution**: Payment provider calls `Execute()` with result
5. **Confirmation/Cancellation**: Webhook handler calls `Confirm()` or `Cancel()`
6. **Callback Redirect**: User is redirected via `RedirectToCallback()`

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Checkout   │────▶│  Payment    │────▶│  Webhook    │
│  (Create)   │     │  Selection  │     │  Execute    │
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

    // Create payment record
    params := sdkapi.CreatePaymentParams{
        Amount:  purchaseReq.Price(),
        Optname: "credit_card",
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

### Webhook Handler

```go
func handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
    // Authenticate webhook
    if err := api.Payments().WebhookAuth(r); err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Parse webhook body
    var params sdkapi.ExecuteParams
    if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

    // Find purchase request
    purchaseReq, err := api.Payments().FindPurchaseRequestByUUID(params.PurchaseUID)
    if err != nil {
        http.Error(w, "Purchase not found", http.StatusNotFound)
        return
    }

    ctx := r.Context()

    if params.Success {
        // Confirm the purchase
        if err := purchaseReq.Confirm(ctx); err != nil {
            api.Logger().Error("Failed to confirm purchase: %v", err)
            http.Error(w, "Failed to confirm", http.StatusInternalServerError)
            return
        }

        // Execute your business logic
        metadata := purchaseReq.Metadata()
        grantAccess(purchaseReq.DeviceID(), metadata)
    } else {
        // Cancel the purchase
        if err := purchaseReq.Cancel(ctx); err != nil {
            api.Logger().Error("Failed to cancel purchase: %v", err)
        }
    }

    w.WriteHeader(http.StatusOK)
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

### Wallet Payment

```go
func handleWalletPayment(w http.ResponseWriter, r *http.Request) {
    purchaseReq, err := api.Payments().GetPurchaseRequest(r)
    if err != nil {
        api.Http().Response().FlashMsg(w, r, "No pending purchase", sdkapi.FlashMsgError)
        return
    }

    ctx := r.Context()

    // Pay with wallet
    err = purchaseReq.PayWithWallet(ctx, purchaseReq.Price())
    if err != nil {
        api.Http().Response().FlashMsg(w, r, "Insufficient wallet balance", sdkapi.FlashMsgError)
        return
    }

    // Check if fully paid
    state, err := purchaseReq.State(ctx)
    if err != nil {
        // handle error
        return
    }

    if state.TotalPayment >= purchaseReq.Price() {
        // Fully paid - confirm purchase
        err = purchaseReq.Confirm(ctx)
        if err != nil {
            // handle error
            return
        }

        api.Http().Response().FlashMsg(w, r, "Purchase complete!", sdkapi.FlashMsgSuccess)
    }

    purchaseReq.RedirectToCallback(w, r)
}
```

---

## Related

- [IPaymentsApi](./payments-api.md) - Payment API for checkout and providers
- [IHttpRouterApi](./http-router-api.md) - Setting up routes for payment handlers
