# IPaymentsApi

The `IPaymentsApi` is used to handle customer payments in Flare Hotspot. It provides methods for registering payment providers, initiating purchases, and managing purchase requests.

## Accessing IPaymentsApi

```go
paymentsApi := api.Payments()
```

---

## IPaymentsApi Methods

The following methods are available in `IPaymentsApi`:

### NewPaymentProvider

Registers a new payment provider. The provider's payment options will become available for customers during checkout.

```go
provider := &MyPaymentProvider{}
api.Payments().NewPaymentProvider(provider)
```

See [IPaymentProvider](#ipaymentprovider) for details on implementing a payment provider.

### Checkout

Creates a purchase request and prompts the user for payment. This method sends an HTTP response and must be the last line in the handler function.

```go
func handlePurchase(w http.ResponseWriter, r *http.Request) {
    purchase := sdkapi.PurchaseRequest{
        Sku:           "wifi_hourly",
        Name:          "1 Hour WiFi Access",
        Description:   "One hour of high-speed internet access",
        Price:         5.00,
        AnyPrice:      false,
        CallbackRoute: "portal:purchase:callback",
        WebHookRoute:  "portal:purchase:webhook",
        Metadata: map[string]string{
            "duration": "3600",
        },
    }

    // This sends HTTP response - must be last line
    api.Payments().Checkout(w, r, purchase)
}
```

### GetPurchaseRequest

Returns the pending purchase request for the client device from the HTTP request.

```go
func handlePayment(w http.ResponseWriter, r *http.Request) {
    purchaseReq, err := api.Payments().GetPurchaseRequest(r)
    if err != nil {
        // No pending purchase or error
        return
    }

    fmt.Printf("Purchase: %s - $%.2f\n", purchaseReq.Name(), purchaseReq.Price())
}
```

### FindPurchaseRequestByUUID

Returns a purchase request by its UUID. Useful for webhook handlers that receive the purchase UUID.

```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    uuid := r.URL.Query().Get("purchase_uuid")

    purchaseReq, err := api.Payments().FindPurchaseRequestByUUID(uuid)
    if err != nil {
        // Purchase not found
        return
    }

    fmt.Printf("Found purchase: %s\n", purchaseReq.Name())
}
```

### FormatCurrency

Formats a float64 amount as a currency string using the current application currency settings.

```go
amount := 29.99
formatted := api.Payments().FormatCurrency(amount)
fmt.Println(formatted) // e.g., "₱29.99" or "$29.99"
```

### ExtractPurchaseData

Extracts and validates purchase data from the request using the `token` query parameter. This method handles both callback requests (GET) and webhook requests (POST).

The token is a JWT signed with the application secret containing the device ID and purchase UUID. It verifies the token and returns the purchase request. Tokens expire after 5 minutes for security.

```go
func handleCallback(w http.ResponseWriter, r *http.Request) {
    // Token is passed as ?token=<jwt> query parameter
    purchaseReq, err := api.Payments().ExtractPurchaseData(r)
    if err != nil {
        http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
        return
    }

    if purchaseReq.IsConfirmed() {
        // Show success page
    }
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    // Same method works for webhooks (POST with ?token=<jwt>)
    purchaseReq, err := api.Payments().ExtractPurchaseData(r)
    if err != nil {
        w.WriteHeader(http.StatusUnauthorized)
        return
    }

    // Process webhook...
}
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

### SupportedCurrency

The following currency constants are available:

```go
const (
    CurrencyPhilippinePeso SupportedCurrency = "PHP"
    CurrencyUsDollar       SupportedCurrency = "USD"
    CurrencyNigerianNaira  SupportedCurrency = "NGN"
)
```

---

## IPaymentProvider

The `IPaymentProvider` interface represents a payment provider that can process payments.

```go
type IPaymentProvider interface {
    // Returns the name of the payment provider
    Name() string

    // Returns available payment options for the current request
    OptionsFactory(r *http.Request) []PaymentOption
}
```

### Implementing a Payment Provider

```go
type MyPaymentProvider struct {
    api sdkapi.IPluginApi
}

func (p *MyPaymentProvider) Name() string {
    return "My Payment Gateway"
}

func (p *MyPaymentProvider) OptionsFactory(r *http.Request) []sdkapi.PaymentOption {
    return []sdkapi.PaymentOption{
        {
            UUID:      sdkutils.Sha1Hash("myplugin:credit_card")[:16],
            Name:      "Credit Card",
            RouteName: "myplugin:payment:credit_card",
            RouteParams: map[string]string{},
        },
        {
            UUID:      sdkutils.Sha1Hash("myplugin:gcash")[:16],
            Name:      "GCash",
            RouteName: "myplugin:payment:gcash",
            RouteParams: map[string]string{},
        },
    }
}

// Register the provider in your plugin's Init function
func Init(api sdkapi.IPluginApi) error {
    provider := &MyPaymentProvider{api: api}
    api.Payments().NewPaymentProvider(provider)
    return nil
}
```

### PaymentOption

The `PaymentOption` struct represents a single payment method offered by a provider:

```go
type PaymentOption struct {
    UUID        string            // Unique, stable identifier (16-char hash, e.g., based on MAC address)
    Name        string            // Display label for the user
    RouteName   string            // Route to handle this payment option
    RouteParams map[string]string // Additional route parameters
}
```

**UUID Best Practices:**
- Generate based on stable device properties (MAC address, serial number, device ID)
- Use consistent hashing (SHA1 truncated to 16 chars) for reproducibility
- Include plugin namespace prefix to prevent collisions (e.g., `"wireless-coinslot:{MAC}"`)
- Never change once assigned to a device - UUID must remain stable even if display name changes

**Example:**
```go
import "github.com/flarehotspot/sdk-utils"

func generatePaymentOptionUUID(macAddress string) string {
    normalized := strings.ToUpper(strings.ReplaceAll(macAddress, ":", ""))
    seed := "wireless-coinslot:" + normalized
    fullHash := sdkutils.Sha1Hash(seed)
    return fullHash[:16] // Truncate to 16 characters
}
```

---

## Usage Examples

### Basic Purchase Flow

```go
// 1. Create a route to initiate purchase
func handleBuyWifi(w http.ResponseWriter, r *http.Request) {
    purchase := sdkapi.PurchaseRequest{
        Sku:           "wifi_1hr",
        Name:          "1 Hour WiFi",
        Description:   "One hour of internet access",
        Price:         10.00,
        CallbackRoute: "myplugin:purchase:callback",
        WebHookRoute:  "myplugin:purchase:webhook",
    }

    api.Payments().Checkout(w, r, purchase)
}

// 2. Handle the webhook (called by payment system)
func handlePurchaseWebhook(w http.ResponseWriter, r *http.Request) {
    // Authenticate the webhook request
    if err := api.Payments().WebhookAuth(r); err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Parse webhook parameters
    var params sdkapi.ExecuteParams
    if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

    // Find the purchase request
    purchaseReq, err := api.Payments().FindPurchaseRequestByUUID(params.PurchaseUID)
    if err != nil {
        http.Error(w, "Purchase not found", http.StatusNotFound)
        return
    }

    ctx := r.Context()

    if params.Success {
        // Confirm the purchase
        if err := purchaseReq.Confirm(ctx); err != nil {
            http.Error(w, "Failed to confirm", http.StatusInternalServerError)
            return
        }

        // Grant access to the user (your business logic)
        grantWifiAccess(purchaseReq.DeviceID(), purchaseReq.Metadata())
    } else {
        // Cancel the purchase
        purchaseReq.Cancel(ctx)
    }

    w.WriteHeader(http.StatusOK)
}

// 3. Handle the callback (user redirect after payment)
func handlePurchaseCallback(w http.ResponseWriter, r *http.Request) {
    purchaseReq, err := api.Payments().GetPurchaseRequest(r)
    if err != nil {
        // Redirect to error page
        return
    }

    if purchaseReq.IsConfirmed() {
        // Show success page
        api.Http().Response().Redirect(w, r, "myplugin:success", nil)
    } else if purchaseReq.IsCancelled() {
        // Show cancelled page
        api.Http().Response().Redirect(w, r, "myplugin:cancelled", nil)
    } else {
        // Still processing
        api.Http().Response().Redirect(w, r, "myplugin:processing", nil)
    }
}
```

### Flexible Pricing (Donations)

```go
func handleDonation(w http.ResponseWriter, r *http.Request) {
    purchase := sdkapi.PurchaseRequest{
        Sku:           "donation",
        Name:          "Support Us",
        Description:   "Make a donation of any amount",
        Price:         0,        // No fixed price
        AnyPrice:      true,     // Allow any amount
        CallbackRoute: "myplugin:donation:callback",
        WebHookRoute:  "myplugin:donation:webhook",
    }

    api.Payments().Checkout(w, r, purchase)
}
```

## Related

- [IPurchaseRequest](./purchase-request.md) - Purchase request interface and lifecycle
- [IHttpRouterApi](./http-router-api.md) - Setting up routes for payment handlers
