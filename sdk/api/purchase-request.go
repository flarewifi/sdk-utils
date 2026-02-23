/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"context"
	"net/http"
	"time"
)

type PurchasePaymentData struct {
	PurchaseID      int64   `json:"purchase_id"`
	TotalPayment    float64 `json:"total_payment"`
	PaymentProvider string  `json:"payment_provider"`
	WalletDebit     float64 `json:"wallet_debit"`
	WalletEndingBal float64 `json:"wallet_ending_bal"`
	WalletRealBal   float64 `json:"wallet_real_bal"`
}

// ExecuteParams holds parameters for executing a purchase webhook.
type ExecuteParams struct {
	Amount  float64 `json:"amount"`
	Success bool    `json:"success"`
	Message string  `json:"message"`
}

// PurchaseRequest represents a purchase to be made by the customer.
type PurchaseRequest struct {
	Sku           string
	Name          string
	Description   string
	Price         float64
	AnyPrice      bool
	CallbackRoute string
	WebHookRoute  string
	Metadata      map[string]string
	Processing    bool
	PaymentUrl    string
}

// CreatePaymentParams holds parameters for creating a payment for a purchase.
type CreatePaymentParams struct {
	Amount       float64
	ProviderUUID string
}

// CreatePurchaseParams holds parameters for creating a purchase programmatically.
// Used for admin-generated purchases where no customer device is involved.
type CreatePurchaseParams struct {
	DeviceID    *int64            // Optional - nil for admin purchases (e.g., voucher batch sales)
	Sku         string            // SKU identifier for the purchase
	Name        string            // Display name of the purchase
	Description string            // Description of the purchase
	Price       float64           // Price of the purchase
	Metadata    map[string]string // Additional metadata
}

// IPurchaseRequest represents a record in purchases table in the database.
type IPurchaseRequest interface {
	// Returns the database ID of the purchase request.
	ID() int64

	// Returns the UUID of the purchase request.
	UUID() string

	// Returns the device ID associated with the purchase.
	DeviceID() int64

	// Returns the SKU of the purchase item.
	Sku() string

	// Returns the name of the purchase item.
	Name() string

	// Returns the description of the purchase item.
	Description() string

	// Returns the price of the purchase item.
	Price() float64

	// Returns true if the purchase request allows any price.
	AnyPrice() bool

	// Returns true if the purchase request has a fixed price.
	IsFixedPrice() bool

	// Returns the wallet debit amount for the purchase.
	WalletDebit() float64

	// Returns the wallet transaction ID if available.
	WalletTxID() *int64

	// Returns the timestamp when the purchase was confirmed.
	ConfirmedAt() *time.Time

	// Returns the timestamp when the purchase was cancelled.
	CancelledAt() *time.Time

	// Returns the reason for cancellation if cancelled.
	CancelledReason() *string

	// Returns the timestamp when the purchase was created.
	CreatedAt() time.Time

	// Returns the callback plugin package name.
	CallbackPluginPkg() string

	// Returns the callback route for the purchase.
	CallbackRoute() string

	// Returns the webhook route for the purchase.
	WebHookRoute() string

	// Returns the metadata associated with the purchase.
	Metadata() map[string]string

	// Returns true if the purchase is confirmed.
	IsConfirmed() bool

	// Returns true if the purchase is cancelled.
	IsCancelled() bool

	// Returns true if the purchase is still processing.
	Processing() bool

	// Returns the payment URL for the purchase.
	PaymentUrl() string

	// Set the processing state and payment URL for the purchase.
	// If paymentUrl is empty, it clears the processing state (sets processing to false).
	// If paymentUrl is provided, it sets processing to true and stores the URL.
	SetProcessing(ctx context.Context, paymentUrl string) error

	// Create a payment for the purchase.
	CreatePayment(ctx context.Context, params CreatePaymentParams) error

	// Pay using the customers wallet.
	// The amount will be debitted from the wallet once the purchase request has been confirmed.
	PayWithWallet(ctx context.Context, amount float64) error

	// Returns the state of the purchase.
	// The state includes the total accumulated payment for the purchase and other important details.
	State(ctx context.Context) (PurchasePaymentData, error)

	// Executes the webhook for the purchase.
	// This will make an internal POST request to the webhook route.
	// The params contain the success status and message to be passed to the webhook handler.
	Execute(ctx context.Context, params ExecuteParams) error

	// Redirects the user to the callback route of the purchase request.
	RedirectToCallback(w http.ResponseWriter, r *http.Request)

	// Confirm the purchase.
	// This must be executed in the purchase webhook handler.
	Confirm(ctx context.Context) error

	// Cancel the purchase.
	// This must be executed in the purchase webhook handler.
	Cancel(ctx context.Context) error

	// UpdateMetadata updates the metadata associated with the purchase.
	// This should be called before Confirm() to ensure metadata is available for sync.
	UpdateMetadata(ctx context.Context, metadata map[string]string) error
}
