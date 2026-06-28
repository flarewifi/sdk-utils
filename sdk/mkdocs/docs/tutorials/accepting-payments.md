# Accept Payments

This guide explains how to implement payment processing in Flarewifi plugins. It covers both product plugins (that create purchase requests) and payment provider plugins (that process payments).

## Payment Flow Diagram

The following diagram illustrates the complete payment flow between a requester plugin (e.g., wifi-hotspot) and a payment provider (e.g., wireless-coinslot):

![Payment Flow Diagram](diagrams/payment-flow.svg)

**Flow Steps:**

1. **Checkout** - wifi-hotspot plugin initiates purchase request
2. **Save Purchase** - Core API saves purchase and shows payment options
3. **SetProcessing** - User selects payment method, provider marks as processing
4. **CreatePayment** - Provider records payment as it's received
5. **Store Payment** - Core API updates payment totals
6. **Execute** - Provider calls `Execute()` with final payment result
7. **In-process Dispatch** - Core invokes the registered `PurchaseExecuteHandler` directly (no HTTP)
8. **ExecuteHandler** - wifi-hotspot handler confirms purchase and creates session
9. **Confirm** - Purchase marked as confirmed in database
10. **Update Status** - Core API updates purchase status
11. **CreateSession** - wifi-hotspot creates session for user
12. **User Connected** - User is connected to internet

## Payment Architecture Overview

Flarewifi uses an in-process purchase execution model:

```
┌─────────────────┐         ┌──────────────────┐         ┌─────────────────┐
│  Product Plugin │         │ Payment Provider │         │  Product Plugin │
│                 │         │                  │         │                 │
│  Creates        │────────▶│  Processes       │────────▶│  Execute        │
│  Purchase       │ Browser │  Payment         │In-proc  │  Handler        │
│  Request        │ Flow    │                  │Dispatch │                 │
│                 │         │                  │         │  - Confirms     │
│                 │         │  Calls           │         │  - Creates      │
│                 │         │  Execute()       │         │    Session      │
│                 │◀────────│                  │         │  - Connects     │
│  Callback       │ Browser │                  │         │    User         │
│  (redirect)     │ Redirect│                  │         │                 │
└─────────────────┘         └──────────────────┘         └─────────────────┘
```

### Key Components

1. **CallbackRoute** - Browser redirect route after payment (user-facing, GET request)
2. **Execute()** - Payment provider calls this to invoke the product plugin's handler in-process
3. **HandlePurchaseExecute()** - Product plugin registers its single execute handler; the core routes to it by the purchase's callback plugin (no name or route needed)
4. **Confirm()** - Execute handler calls this to mark the purchase complete

## Creating a Product Plugin

Product plugins create purchase requests for users to buy (e.g., WiFi sessions, vouchers, downloads).

### Step 1: Create Purchase Request

```go
package portal

import (
    "net/http"
    sdkapi "sdk/api"
)

func PurchaseWifiSession(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        p := sdkapi.PurchaseRequest{
            Sku:           "wifi-connection",
            Name:          "WiFi Connection",
            Description:   "Internet access for 60 minutes",
            Price:         5.00,
            AnyPrice:      false,
            CallbackRoute: "purchase.wifi.callback",
            Metadata: map[string]string{
                "time_mins": "60",
                "data_mb":   "1024",
            },
        }
        api.Payments().Checkout(w, r, p)
    }
}
```

### Step 2: Register Execute Handler

Register an in-process handler in your plugin's `Init` function. Each plugin has **one** execute handler — the core routes to it by the purchase's callback plugin, so there's no name or route to match. When a payment provider calls `purchase.Execute()`, the core invokes your handler directly — no HTTP request, no JWT verification needed. Branch on `purchase.Sku()` (or `purchase.Metadata()`) inside the handler to fulfil different purchase types.

```go
package main

import (
    "context"
    "fmt"

    sdkapi "sdk/api"
    sdkutils "github.com/flarewifi/sdk-utils"
)

func Init(api sdkapi.IPluginApi) error {
    // Register routes
    SetRoutes(api)

    // Register this plugin's in-process execute handler (routed by callback plugin)
    api.Payments().HandlePurchaseExecute(func(ctx context.Context, purchase sdkapi.IPurchaseRequest, params sdkapi.ExecuteParams) error {

        // Handle payment failure
        if !params.Success {
            api.Logger().Error(api.Translate("error", "Payment failed: <% .msg %>", "msg", params.Message))
            purchase.Cancel(ctx)
            return nil
        }

        // Find client device
        clnt, err := api.SessionsMgr().FindClientById(ctx, purchase.DeviceID())
        if err != nil {
            return fmt.Errorf("client not found: %w", err)
        }

        // Confirm purchase FIRST (critical!)
        if err = purchase.Confirm(ctx); err != nil {
            return fmt.Errorf("failed to confirm purchase: %w", err)
        }

        // Create WiFi session
        session, err := api.SessionsMgr().CreateSession(ctx, sdkapi.CreateSessionParams{
            UUID:           sdkutils.NewUUID(),
            DevId:          clnt.ID(),
            Type:           sdkapi.SessionTypeTime,
            TimeSecs:       3600, // 60 minutes
            DataMb:         1024,
            DownMbits:      10,
            UpMbits:        10,
            UseGlobalSpeed: false,
        })
        if err != nil {
            // Rollback: purchase was already confirmed, but we failed to create the session.
            // Log the error so the operator can manually resolve.
            api.Logger().Error(api.Translate("error", "Failed to create session after confirm: <% .err %>", "err", err))
            return fmt.Errorf("failed to create session: %w", err)
        }

        api.Logger().Info(api.Translate("info", "Session created: <% .id %>", "id", session.ID()))

        // Auto-connect user to internet
        if !api.SessionsMgr().IsConnected(clnt) {
            msg := api.Translate("success", "Payment successful! Connecting to internet...")
            if err = api.SessionsMgr().Connect(ctx, clnt, msg); err != nil {
                // Not fatal — session exists, user can reconnect
                api.Logger().Error(api.Translate("error", "Auto-connect failed: <% .err %>", "err", err))
            }
        }

        return nil
    })

    return nil
}
```

### Step 3: Create Callback Handler

The callback route is where the browser redirects after payment. It is optional — use it to show a success or failure page.

```go
func PurchaseCallback(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        purchaseReq, err := api.Payments().GetPurchaseRequest(r)
        if err != nil {
            http.Redirect(w, r, "/", http.StatusSeeOther)
            return
        }

        res := api.Http().Response()

        if purchaseReq.IsConfirmed() {
            res.FlashMsg(w, r, api.Translate("success", "Payment successful!"), sdkapi.FlashMsgSuccess)
        } else if purchaseReq.IsCancelled() {
            res.FlashMsg(w, r, api.Translate("error", "Payment was cancelled."), sdkapi.FlashMsgError)
        }

        http.Redirect(w, r, "/", http.StatusSeeOther)
    }
}
```

### Step 4: Register Routes

Only the checkout and callback routes need to be registered. The execute handler is registered in `Init` via `HandlePurchaseExecute`, not as an HTTP route.

```go
package routes

import (
    sdkapi "sdk/api"
    controllers "yourplugin/app/controllers/portal"
)

func SetRoutes(api sdkapi.IPluginApi) {
    router := api.Http().Router().PluginRouter()

    router.Group("/purchase", func(subrouter sdkapi.IHttpRouterInstance) {
        subrouter.Get("/wifi", controllers.PurchaseWifiSession(api)).Name("purchase.wifi")
        subrouter.Get("/callback", controllers.PurchaseCallback(api)).Name("purchase.wifi.callback")
        // The execute handler is registered in-process via HandlePurchaseExecute
        // and routed by this plugin's package — no HTTP route needed.
    })
}
```

## Creating a Payment Provider Plugin

Payment provider plugins process payments (e.g., wireless-coinslot, PayStack, Stripe).

### Step 1: Implement Payment Provider Interface

```go
package src

import (
    "net/http"
    "strings"

    sdkapi "sdk/api"
    sdkutils "github.com/flarewifi/sdk-utils"
)

type MyPaymentProvider struct {
    name string
    api  sdkapi.IPluginApi
}

func NewPaymentProvider(api sdkapi.IPluginApi) *MyPaymentProvider {
    return &MyPaymentProvider{
        name: api.Translate("label", "My Payment Method"),
        api:  api,
    }
}

func (self *MyPaymentProvider) Name() string {
    return self.name
}

func (self *MyPaymentProvider) OptionsFactory(r *http.Request) []sdkapi.PaymentOption {
    return []sdkapi.PaymentOption{
        {
            UUID:        generatePaymentOptionUUID("AA:BB:CC:DD:EE:FF"),
            Name:        "Main Entrance Coinslot",
            RouteName:   "payments.coinslot",
            RouteParams: map[string]string{"id": "coinslot-1"},
        },
    }
}

// generatePaymentOptionUUID creates a stable 16-char UUID from a device MAC address.
func generatePaymentOptionUUID(macAddress string) string {
    normalized := strings.ToUpper(strings.ReplaceAll(macAddress, ":", ""))
    seed := "your-plugin-name:" + normalized
    return sdkutils.Sha1Hash(seed)[:16]
}
```

### Step 2: Create Payment Handler

This handler collects payment from the user and calls `purchase.Execute()` with the result. `ExecuteParams` carries only the payment outcome — the purchase record is already known from the method receiver.

```go
func ProcessPayment(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        res := api.Http().Response()
        ctx := r.Context()

        purchase, err := api.Payments().GetPurchaseRequest(r)
        if err != nil {
            res.Error(w, r, err, 500)
            return
        }

        // ... collect payment from user (coins, API call, etc.) ...
        amountPaid := 31.00

        // Execute triggers the product plugin's registered handler in-process.
        // Only pass Amount, Success, and Message — no DeviceID or PurchaseUID needed.
        err = purchase.Execute(ctx, sdkapi.ExecuteParams{
            Amount:  amountPaid,
            Success: true,
            Message: "Payment successful",
        })
        if err != nil {
            api.Logger().Error(api.Translate("error", "Execute failed: <% .err %>", "err", err))
            res.Error(w, r, err, 500)
            return
        }

        res.Json(w, r, map[string]interface{}{
            "success": true,
            "message": "Payment processed",
        }, 200)
    }
}
```

### Step 3: Register Payment Provider

```go
func Init(api sdkapi.IPluginApi) error {
    provider := NewPaymentProvider(api)
    api.Payments().NewPaymentProvider(provider)

    SetRoutes(api)
    return nil
}
```

## Important Concepts

### CallbackRoute vs the execute handler

A purchase has two distinct fulfilment paths. `CallbackRoute` is a field on the
`PurchaseRequest`; the execute handler is registered separately via
`HandlePurchaseExecute` and is **not** referenced by any field — the core routes
to it by the purchase's callback plugin.

| Aspect | CallbackRoute (field) | Execute handler (`HandlePurchaseExecute`) |
|--------|----------------------|-------------------------------------------|
| **What it is** | A named browser route on the `PurchaseRequest` | The plugin's in-process fulfilment function |
| **Transport** | HTTP GET (browser) | In-process function call (no HTTP) |
| **Routing** | Mux route name | Matched by the purchase's callback plugin |
| **Triggers** | Browser redirect after `Execute()` returns | `purchase.Execute()` called by provider |
| **Use case** | Show success/failure page | Confirm purchase, create session |
| **Required** | Optional | Required if you need to grant access |

### Payment Flow Sequence

1. **User initiates purchase** → Product plugin creates `PurchaseRequest` via `Checkout()`
2. **User selects payment method** → Payment system shows available providers
3. **User makes payment** → Payment provider collects payment
4. **Provider calls `Execute()`** → Core dispatches in-process to the registered handler
5. **Handler confirms purchase** → Calls `purchase.Confirm(ctx)`
6. **Handler creates session** → User gets their product/service
7. **Handler auto-connects user** → User gets internet access
8. **`Execute()` returns** → Payment provider shows success to user
9. **Browser redirects** → User returns to `CallbackRoute` (optional)

### Critical Implementation Details

#### 1. Confirm Purchase BEFORE Creating Session

```go
// CORRECT ORDER
if err = purchase.Confirm(ctx); err != nil {
    return err
}
session, err := api.SessionsMgr().CreateSession(ctx, ...)

// WRONG ORDER — purchase stays unconfirmed if confirm fails later
session, err := api.SessionsMgr().CreateSession(ctx, ...)
if err = purchase.Confirm(ctx); err != nil {
    return err
}
```

#### 2. Handle Payment Failures

```go
if !params.Success {
    api.Logger().Error(api.Translate("error", "Payment failed: <% .msg %>", "msg", params.Message))
    purchase.Cancel(ctx)
    return nil // not an error — failure is expected, not exceptional
}
```

#### 3. Always Check Execute() Return Value

```go
// CORRECT
err = purchase.Execute(ctx, params)
if err != nil {
    api.Logger().Error(api.Translate("error", "Execute failed: <% .err %>", "err", err))
    return
}

// WRONG — silently ignores errors
purchase.Execute(ctx, params)
```

#### 4. Always Generate Session UUID

```go
import sdkutils "github.com/flarewifi/sdk-utils"

session, err := api.SessionsMgr().CreateSession(ctx, sdkapi.CreateSessionParams{
    UUID:           sdkutils.NewUUID(), // Required!
    DevId:          clnt.ID(),
    Type:           sdkapi.SessionTypeTime,
    TimeSecs:       3600,
    DataMb:         1024,
    DownMbits:      10,
    UpMbits:        10,
    UseGlobalSpeed: false,
})
```

Without a UUID, `CreateSession` returns an error: `"session UUID is required"`.

## Common Patterns

### Variable Pricing (AnyPrice)

```go
p := sdkapi.PurchaseRequest{
    Sku:           "wifi-donation",
    Name:          "WiFi Access",
    Description:   "Pay what you want",
    AnyPrice:      true,
    Price:         0,
    CallbackRoute: "purchase.wifi.callback",
}
```

In your execute handler, calculate session time based on the paid amount:

```go
sessionSeconds := calculateSessionTime(params.Amount, rateConfig)
```

### Fixed Pricing

```go
p := sdkapi.PurchaseRequest{
    Sku:           "wifi-1hour",
    Name:          "WiFi 1 Hour",
    Description:   "Internet access for 60 minutes",
    AnyPrice:      false,
    Price:         5.00,
    CallbackRoute: "purchase.wifi.callback",
}
```

### Storing Custom Data

Use the `Metadata` field to store custom information:

```go
p := sdkapi.PurchaseRequest{
    // ...
    Metadata: map[string]string{
        "time_mins":  "120",
        "data_mb":    "2048",
        "rate_index": "2",
    },
}
```

Retrieve in the execute handler:

```go
metadata := purchase.Metadata()
timeMins := metadata["time_mins"]
```

## Testing Your Implementation

### 1. Verify Execute Handler Is Registered

Enable debug logging and watch for the dispatch:

```bash
docker logs flarewifi-app-1 -f 2>&1 | grep -E "(Execute|Confirm|Session)"
```

Expected log sequence:

```
[Provider] Executing purchase with amount 10.00
[Product] Purchase confirmed
[Product] Session created: 42
[Product] Device connected
```

### 2. Test Payment Flow End-to-End

Use the Playwright MCP to navigate through the portal:

1. Navigate to `http://localhost:3000` as a client device
2. Select a WiFi plan → payment options appear
3. Trigger payment from the provider plugin
4. Verify session is created and user is connected

### 3. Verify Purchase Completion

After payment, check that:
- Purchase status is `confirmed` in the database
- User has an active session
- User is connected to the internet

## Troubleshooting

### Purchase Stays in "Processing" State

**Cause:** Execute handler didn't call `purchase.Confirm(ctx)`.

**Fix:** Ensure your handler calls `Confirm()` before returning:

```go
if err = purchase.Confirm(ctx); err != nil {
    return fmt.Errorf("failed to confirm purchase: %w", err)
}
```

### "No payment handler is registered to receive the payment"

**Cause:** The product plugin never registered an execute handler, so the core
has nothing to dispatch to when the provider calls `Execute()`.

**Fix:** Call `HandlePurchaseExecute` in your plugin's `Init` (it takes only the
handler — the core routes to it by your plugin package):

```go
func Init(api sdkapi.IPluginApi) error {
    api.Payments().HandlePurchaseExecute(func(ctx context.Context, purchase sdkapi.IPurchaseRequest, params sdkapi.ExecuteParams) error {
        // confirm purchase, create session, etc.
        return nil
    })
    return nil
}
```

### Execute() Returns Error

**Cause:** Multiple possible issues — handler panicked, handler returned an error, or no handler was registered for the callback plugin.

**Fix:** Check logs for the execute handler:

```bash
docker logs flarewifi-app-1 -f 2>&1 | grep -iE "(execute|handler|dispatch)"
```

Also verify `HandlePurchaseExecute` is actually called during `Init`.

### User Not Auto-Connected

**Cause:** Forgot to call `api.SessionsMgr().Connect()` in the execute handler.

**Fix:**

```go
if !api.SessionsMgr().IsConnected(clnt) {
    api.SessionsMgr().Connect(ctx, clnt, api.Translate("success", "Payment successful!"))
}
```

### "Session UUID is required"

**Cause:** `CreateSession` called without `UUID`.

**Fix:** Always provide a UUID:

```go
import sdkutils "github.com/flarewifi/sdk-utils"

sdkapi.CreateSessionParams{
    UUID: sdkutils.NewUUID(), // required
    // ...
}
```

## API Reference

### PurchaseRequest Struct

```go
type PurchaseRequest struct {
    Sku           string            // Unique identifier for this product type
    Name          string            // Display name
    Description   string            // Short description
    Price         float64           // Price (ignored if AnyPrice is true)
    AnyPrice      bool              // Allow variable pricing
    CallbackRoute string            // Browser redirect route (optional)
    Metadata      map[string]string // Custom data
}
```

### ExecuteParams Struct

```go
type ExecuteParams struct {
    Amount  float64 // Amount paid
    Success bool    // Whether payment was successful
    Message string  // Status message (used for error reason on failure)
}
```

### IPurchaseRequest Interface

Key methods:

```go
// Get purchase details
ID() int64
UUID() string
DeviceID() int64
Price() float64
AnyPrice() bool
Metadata() map[string]string

// In-process execute (payment provider calls this)
Execute(ctx context.Context, params ExecuteParams) error

// Confirm purchase (execute handler calls this after successful payment)
Confirm(ctx context.Context) error

// Cancel purchase (execute handler calls this on payment failure)
Cancel(ctx context.Context) error

// Check status
IsConfirmed() bool
IsCancelled() bool
Processing() bool
```

### CreateSessionParams Struct

```go
type CreateSessionParams struct {
    UUID           string      // Required — use sdkutils.NewUUID()
    DevId          int64       // Client device ID
    Type           SessionType // "time", "data", or "time-or-data"
    TimeSecs       int         // Session duration in seconds
    DataMb         float64     // Data allowance in megabytes
    ExpDays        *int        // Expiry in days (nil = no expiry)
    DownMbits      int         // Download speed limit in Mbps
    UpMbits        int         // Upload speed limit in Mbps
    UseGlobalSpeed bool        // Use global speed settings instead of DownMbits/UpMbits
}
```

## Best Practices

1. **Register handler before routes** — call `HandlePurchaseExecute` early in `Init`
2. **One handler per plugin** — branch on `purchase.Sku()`/`Metadata()` to fulfil different purchase types
3. **Confirm purchases before creating sessions** — ensures data consistency
4. **Handle payment failures gracefully** — cancel purchases, log errors
5. **Log everything** — makes debugging much easier
6. **Return errors from the handler** — the core logs and propagates them back to `Execute()`'s caller
7. **Auto-connect users** — better user experience; failure to connect is non-fatal after session creation
8. **Use `sdkutils.NewUUID()`** — always generate UUIDs for sessions

## Summary

Payment processing in Flarewifi uses an in-process execution model:

1. **Product plugins** create `PurchaseRequest` values with an optional `CallbackRoute`
2. **Product plugins** register one `PurchaseExecuteHandler` via `api.Payments().HandlePurchaseExecute(handler)`
3. **Payment providers** collect payment and call `purchase.Execute(ctx, params)`
4. **`Execute()`** looks up the registered handler by the purchase's callback plugin and invokes it in-process
5. **The handler** confirms the purchase, creates the session, and connects the user
6. **Browser** optionally redirects to `CallbackRoute`

The key advantage of this model: no HTTP round-trip, no JWT authentication overhead, and no cross-goroutine contention on the single database connection.
