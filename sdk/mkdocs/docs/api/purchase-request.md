# IPurchaseRequest

The `IPurchaseRequest` interface represents a purchase request in the Flare Hotspot payment system. It provides methods to manage the lifecycle of a purchase, including creating payments, handling wallet payments, and confirming or canceling purchases.

Purchase requests are typically created through the [IPaymentsApi](./payments-api.md) and obtained from HTTP requests using `api.Payments().GetPurchaseRequest(r)`.

## IPurchaseRequest Methods

The following methods are available in `IPurchaseRequest`:

### Id

Returns the database ID of the purchase request.

```go
id := purchaseRequest.Id()
fmt.Printf("Purchase ID: %d\n", id)
```

### Uid

Returns the unique identifier (UID) of the purchase request.

```go
uid := purchaseRequest.Uid()
fmt.Printf("Purchase UID: %s\n", uid)
```

### Name

Returns the name of the purchase item.

```go
name := purchaseRequest.Name()
fmt.Printf("Item: %s\n", name)
```

### Price

Returns the price of the purchase item.

```go
price := purchaseRequest.Price()
fmt.Printf("Price: $%.2f\n", price)
```

### IsFixedPrice

Returns `true` if the purchase request has a fixed price, `false` if it allows any price (donation-style).

```go
if purchaseRequest.IsFixedPrice() {
    fmt.Println("Fixed price purchase")
} else {
    fmt.Println("Flexible price purchase")
}
```

### CreatePayment

Creates a payment for the purchase request. This initiates the payment process.

```go
params := CreatePaymentParams{
    Amount:  29.99,
    Optname: "credit_card",
}

err := purchaseRequest.CreatePayment(tx, r.Context(), params)
if err != nil {
    // handle error
}
```

### PayWithWallet

Pays for the purchase using the customer's wallet balance. The amount is debited from the wallet when the purchase is confirmed.

```go
amount := 10.00
err := purchaseRequest.PayWithWallet(tx, r.Context(), amount)
if err != nil {
    // handle insufficient funds or other error
}
```

### State

Returns the current state of the purchase, including payment totals and wallet information.

```go
state, err := purchaseRequest.State(tx, r.Context())
if err != nil {
    // handle error
}

fmt.Printf("Purchase ID: %d\n", state.PurchaseID)
fmt.Printf("Total Payment: $%.2f\n", state.TotalPayment)
fmt.Printf("Wallet Debit: $%.2f\n", state.WalletDebit)
fmt.Printf("Ending Balance: $%.2f\n", state.WalletEndingBal)
```

### Execute

Executes the payment for the purchase. This redirects the user to the payment provider or callback URL.

```go
purchaseRequest.Execute(w, r)
// This will redirect the user to complete the payment
```

### Confirm

Confirms the purchase after successful payment. This should be called in the purchase callback handler.

```go
err := purchaseRequest.Confirm(tx, r.Context())
if err != nil {
    // handle error
}
// Purchase is now complete
```

### Cancel

Cancels the purchase. This should be called when payment fails or is canceled.

```go
err := purchaseRequest.Cancel(tx, r.Context())
if err != nil {
    // handle error
}
// Purchase has been canceled
```

## Types

### PurchaseRequest

The `PurchaseRequest` struct represents the initial purchase request data:

```go
type PurchaseRequest struct {
    Sku           string            // Stock keeping unit identifier
    Name          string            // Display name of the item
    Description   string            // Description of the item
    Price         float64           // Price of the item
    AnyPrice      bool              // Whether the item allows flexible pricing
    CallbackRoute string            // Route to redirect after payment
    Metadata      map[string]string // Additional metadata
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
    Optname string  // Payment option name (e.g., "credit_card", "paypal")
}
```

## Usage Examples

### Creating a Purchase Request

```go
// Create a purchase request through the payments API
purchaseReq := PurchaseRequest{
    Sku:           "premium_monthly",
    Name:          "Premium Monthly Subscription",
    Description:   "Access to premium features for one month",
    Price:         9.99,
    AnyPrice:      false,
    CallbackRoute: "portal:payment_callback",
    Metadata: map[string]string{
        "user_id": "12345",
        "plan":    "monthly",
    },
}

// This would typically be done through IPaymentsApi.Checkout
```

### Handling Purchase Flow

```go
func initiatePurchase(w http.ResponseWriter, r *http.Request) {
    // Get or create purchase request
    purchaseRequest := getPurchaseRequest(r) // Your implementation

    // Create payment
    params := CreatePaymentParams{
        Amount:  purchaseRequest.Price(),
        Optname: "stripe", // or other payment provider
    }

    tx, err := api.SqlDB().BeginTx(r.Context(), nil)
    if err != nil {
        // handle error
        return
    }
    defer tx.Rollback()

    err = purchaseRequest.CreatePayment(tx, r.Context(), params)
    if err != nil {
        // handle error
        return
    }

    // Execute the purchase (redirects to payment)
    purchaseRequest.Execute(w, r)
}

func handlePaymentCallback(w http.ResponseWriter, r *http.Request) {
    // Get the purchase request from the request
    purchaseRequest, err := api.Payments().GetPurchaseRequest(r)
    if err != nil {
        // handle error
        return
    }

    tx, err := api.SqlDB().BeginTx(r.Context(), nil)
    if err != nil {
        // handle error
        return
    }
    defer tx.Rollback()

    // Check payment status and confirm or cancel
    if paymentSuccessful(r) {
        err = purchaseRequest.Confirm(tx, r.Context())
        if err != nil {
            // handle error
            return
        }

        // Redirect to success page
        api.Http().Response().Redirect(w, r, "portal:payment_success")
    } else {
        err = purchaseRequest.Cancel(tx, r.Context())
        if err != nil {
            // handle error
            return
        }

        // Redirect to failure page
        api.Http().Response().Redirect(w, r, "portal:payment_failed")
    }

    tx.Commit()
}
```

### Wallet Payment

```go
func payWithWallet(w http.ResponseWriter, r *http.Request) {
    purchaseRequest := getPurchaseRequest(r)

    tx, err := api.SqlDB().BeginTx(r.Context(), nil)
    if err != nil {
        // handle error
        return
    }
    defer tx.Rollback()

    // Pay with wallet
    amount := purchaseRequest.Price()
    err = purchaseRequest.PayWithWallet(tx, r.Context(), amount)
    if err != nil {
        // handle insufficient funds
        api.Http().Response().Redirect(w, r, "portal:insufficient_funds")
        return
    }

    // Confirm the purchase
    err = purchaseRequest.Confirm(tx, r.Context())
    if err != nil {
        // handle error
        return
    }

    tx.Commit()

    // Redirect to success
    api.Http().Response().Redirect(w, r, "portal:purchase_success")
}
```

## Purchase Lifecycle

1. **Create**: Purchase request is created with item details
2. **Payment**: Payment is initiated via `CreatePayment()` or `PayWithWallet()`
3. **Execute**: User is redirected to payment provider via `Execute()`
4. **Callback**: Payment provider calls back to confirm/cancel via `Confirm()` or `Cancel()`
5. **Complete**: Purchase is finalized and user gains access to purchased item

The purchase state can be checked at any time using the `State()` method to monitor payment progress.</content>
<parameter name="filePath">/Users/adonesp/Projects/flarehotspot/sdk/mkdocs/docs/api/purchase-request.md