# IInAppPurchasesApi

The `IInAppPurchasesApi` provides methods to handle in-app purchases and subscriptions in the Flarewifi system. It allows you to verify purchases, check subscriptions, and protect routes that require payment.

To get an instance of `IInAppPurchasesApi`:

```go
inAppPurchases := api.InAppPurchases()
fmt.Println(inAppPurchases) // IInAppPurchasesApi
```

## IInAppPurchasesApi Methods

The following methods are available in `IInAppPurchasesApi`:

### VerifyPurchase

Verifies if the user has already purchased the specified item.

```go
item := InAppCheckoutItem{
    ProductId: "premium_feature_1",
}

err := api.InAppPurchases().VerifyPurchase(item)
if err != nil {
    // User has not purchased this item or verification failed
    fmt.Println("Purchase required")
} else {
    // User has purchased this item
    fmt.Println("Purchase verified")
}
```

### VerifySubscription

Verifies if the user has an active subscription to the specified item.

```go
subscription := InAppSubscriptionItem{
    PlanId: "monthly_premium",
}

err := api.InAppPurchases().VerifySubscription(subscription)
if err != nil {
    // User does not have an active subscription
    fmt.Println("Subscription required")
} else {
    // User has an active subscription
    fmt.Println("Subscription verified")
}
```

### PurchaseGuardMiddleware

Returns a middleware that redirects users to the purchase page if they haven't purchased the specified item. This middleware should be applied to routes that require a purchase.

```go
item := InAppCheckoutItem{
    ProductId: "premium_feature_1",
}

purchaseMiddleware := api.InAppPurchases().PurchaseGuardMiddleware(item)

// Apply the middleware to protect routes
router := api.Http().Router().PluginRouter()
router.Get("/premium-feature", func(w http.ResponseWriter, r *http.Request) {
    // This code only runs if the user has purchased the item
    fmt.Println("Accessing premium feature")
}, purchaseMiddleware)
```

### SubscriptionGuardMiddleware

Returns a middleware that redirects users to the subscription page if they don't have an active subscription to the specified item. This middleware should be applied to routes that require a subscription.

```go
subscription := InAppSubscriptionItem{
    PlanId: "monthly_premium",
}

subscriptionMiddleware := api.InAppPurchases().SubscriptionGuardMiddleware(subscription)

// Apply the middleware to protect routes
router := api.Http().Router().PluginRouter()
router.Get("/premium-content", func(w http.ResponseWriter, r *http.Request) {
    // This code only runs if the user has an active subscription
    fmt.Println("Accessing premium content")
}, subscriptionMiddleware)
```

## Types

### InAppCheckoutItem

The `InAppCheckoutItem` struct represents a purchasable item:

```go
type InAppCheckoutItem struct {
    ProductId string // The product ID from your payment provider
}
```

### InAppSubscriptionItem

The `InAppSubscriptionItem` struct represents a subscription item:

```go
type InAppSubscriptionItem struct {
    PlanId string // The subscription plan ID from your payment provider
}
```

## Usage Examples

### Protecting Premium Features

```go
func setupPremiumRoutes() {
    router := api.Http().Router().PluginRouter()

    // Define the premium item
    premiumFeature := InAppCheckoutItem{
        ProductId: "premium_feature_pack",
    }

    // Get the purchase guard middleware
    purchaseGuard := api.InAppPurchases().PurchaseGuardMiddleware(premiumFeature)

    // Protect premium routes
    router.Get("/premium/dashboard", showPremiumDashboard, purchaseGuard)
    router.Get("/premium/reports", showPremiumReports, purchaseGuard)
    router.Post("/premium/export", exportPremiumData, purchaseGuard)
}

func showPremiumDashboard(w http.ResponseWriter, r *http.Request) {
    // This handler only executes if the user has purchased the premium feature
    api.Http().Response().PortalView(w, r, sdkapi.ViewPage{
        PageContent: views.PremiumDashboard(),
    })
}
```

### Subscription-Based Content

```go
func setupSubscriptionRoutes() {
    router := api.Http().Router().PluginRouter()

    // Define the subscription plan
    premiumPlan := InAppSubscriptionItem{
        PlanId: "premium_monthly",
    }

    // Get the subscription guard middleware
    subscriptionGuard := api.InAppPurchases().SubscriptionGuardMiddleware(premiumPlan)

    // Protect subscription-based routes
    router.Get("/content/movies", streamMovies, subscriptionGuard)
    router.Get("/content/music", streamMusic, subscriptionGuard)
    router.Get("/content/articles", viewArticles, subscriptionGuard)
}
```

### Checking Purchase Status

```go
func checkUserAccess(userId int64, productId string) bool {
    item := InAppCheckoutItem{ProductId: productId}

    // Note: This is a simplified example. In practice, you might need
    // to check purchases for a specific user context
    err := api.InAppPurchases().VerifyPurchase(item)
    return err == nil
}

// Usage in handler
func premiumHandler(w http.ResponseWriter, r *http.Request) {
    if !checkUserAccess(getCurrentUserId(r), "premium_access") {
        api.Http().Response().Redirect(w, r, "portal:upgrade")
        return
    }

    // Show premium content
    api.Http().Response().PortalView(w, r, sdkapi.ViewPage{
        PageContent: views.PremiumContent(),
    })
}
```

## Integration Notes

- The actual purchase/subscription verification is handled by the underlying payment system
- Product IDs and Plan IDs must match those configured in your payment provider (Stripe, PayPal, etc.)
- The middleware automatically redirects users to appropriate payment pages when access is denied
- Purchase and subscription states are typically cached for performance</content>
<parameter name="filePath">/Users/adonesp/Projects/flarehotspot/sdk/mkdocs/docs/api/inapp-purchase-api.md