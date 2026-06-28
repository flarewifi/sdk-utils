# IInAppPurchasesApi

!!! danger "Not yet implemented"
    `IInAppPurchasesApi` is a **draft interface**. The methods exist so plugins can
    compile against them, but the core implementation is currently a **no-op stub**:

    - `CheckOneTimePurchase` and `CheckSubscription` always return an empty status
      (zero value) and a `nil` error — they never report a real purchase/subscription.
    - `PurchaseGuardMiddleware` and `SubscriptionGuardMiddleware` return a
      **pass-through** middleware that lets *every* request through — they do **not**
      gate access yet.

    Do **not** rely on this API to protect paid features. The method set and types
    below may still change before it ships. For working payments today, use
    [IPaymentsApi](./payments-api.md) and [IPurchaseRequest](./purchase-request.md).

The `IInAppPurchasesApi` is intended to let plugins sell one-time products and
subscriptions and gate routes behind them, without each plugin re-implementing the
checkout/verification flow.

To get an instance of `IInAppPurchasesApi`:

```go
inAppPurchases := api.InAppPurchases()
```

## IInAppPurchasesApi Methods

The following methods are defined on `IInAppPurchasesApi`. (See the warning above —
they are not functional yet.)

### CheckOneTimePurchase

Checks whether the current customer has paid for a one-time product, identified by
its `productID`.

```go
status, err := api.InAppPurchases().CheckOneTimePurchase("premium_feature_1")
if err != nil {
    // verification failed
    return
}

if status.Status == sdkapi.InAppPurchaseStatusPaid {
    // customer owns this product
} else {
    // unpaid — status.Message may explain why
}
```

### CheckSubscription

Checks whether the current customer has an active subscription to the plan
identified by its `planID`.

```go
status, err := api.InAppPurchases().CheckSubscription("premium_monthly")
if err != nil {
    // verification failed
    return
}

if status.Status == sdkapi.InAppPurchaseStatusPaid {
    // active subscription
}
```

### PurchaseGuardMiddleware

Returns an HTTP middleware that should redirect the customer to the purchase page
when they have not paid for the given one-time product. Apply it to routes that
require a purchase.

```go
guard := api.InAppPurchases().PurchaseGuardMiddleware(sdkapi.InAppOneTimePurchaseStatus{
    ProductID: "premium_feature_1",
})

router := api.Http().Router().HttpRouter(nil)
router.Get("/premium-feature", func(w http.ResponseWriter, r *http.Request) {
    // Intended: only runs once the customer owns the product.
    // NOTE: currently the guard is a no-op and always allows access.
}, guard)
```

### SubscriptionGuardMiddleware

Returns an HTTP middleware that should redirect the customer to the subscription
page when they do not have an active subscription. Apply it to routes that require
a subscription.

```go
guard := api.InAppPurchases().SubscriptionGuardMiddleware(sdkapi.InAppSubscription{
    PlanId:                "premium_monthly",
    SubscriptionFrequency: sdkapi.InAppSubscriptionMonthly,
})

router := api.Http().Router().HttpRouter(nil)
router.Get("/premium-content", func(w http.ResponseWriter, r *http.Request) {
    // Intended: only runs for active subscribers.
    // NOTE: currently the guard is a no-op and always allows access.
}, guard)
```

## Types

### InAppPurchaseStatus

A string enum describing payment state, used by both status structs.

```go
type InAppPurchaseStatus string

const (
    InAppPurchaseStatusUnpaid InAppPurchaseStatus = "unpaid"
    InAppPurchaseStatusPaid   InAppPurchaseStatus = "paid"
)
```

### InAppOneTimePurchaseStatus

Returned by `CheckOneTimePurchase` (and passed to `PurchaseGuardMiddleware`).

```go
type InAppOneTimePurchaseStatus struct {
    ProductID string             // The product ID, from your developer console dashboard
    Status    InAppPurchaseStatus // "paid" or "unpaid"
    Message   string             // Human-readable detail (e.g. why it is unpaid)
}
```

### InAppOneTimePurchase

Describes a purchasable one-time product and its display price.

```go
type InAppOneTimePurchase struct {
    ProductID       string
    DisplayPrice    float64
    DisplayCurrency string
}
```

### InAppSubscriptionFrequency

A string enum for subscription billing cadence.

```go
type InAppSubscriptionFrequency string

const (
    InAppSubscriptionMonthly InAppSubscriptionFrequency = "monthly"
    InAppSubscriptionYearly  InAppSubscriptionFrequency = "yearly"
)
```

### InAppSubscriptionStatus

Returned by `CheckSubscription`.

```go
type InAppSubscriptionStatus struct {
    PlanId  string
    Status  InAppPurchaseStatus // "paid" (active) or "unpaid"
    Message string
}
```

### InAppSubscription

Describes a subscription plan, passed to `SubscriptionGuardMiddleware`.

```go
type InAppSubscription struct {
    PlanId                string
    SubscriptionFrequency InAppSubscriptionFrequency
    DisplayPrice          float64
    DisplayCurrency       float64
}
```

## Related

- [IPaymentsApi](./payments-api.md) — the implemented payments/checkout API
- [IPurchaseRequest](./purchase-request.md) — manage the lifecycle of a purchase
- [Accept Payments](../tutorials/accepting-payments.md) — end-to-end payment tutorial
