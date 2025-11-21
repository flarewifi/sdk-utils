/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"context"
	"database/sql"
	"net/http"
)

type PurchaseState struct {
	PurchaseID      int64   `json:"purchase_id"`
	TotalPayment    float64 `json:"total_payment"`
	WalletDebit     float64 `json:"wallet_debit"`
	WalletEndingBal float64 `json:"wallet_ending_bal"`
	WalletRealBal   float64 `json:"wallet_real_bal"`
}

// ExecuteParams holds parameters for executing a purchase webhook.
type ExecuteParams struct {
	DeviceID    int64   `json:"device_id"`
	PurchaseUID string  `json:"purchase_uid"`
	Amount      float64 `json:"amount"`
	Success     bool    `json:"success"`
	Message     string  `json:"message"`
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
}

// CreatePaymentParams holds parameters for creating a payment for a purchase.
type CreatePaymentParams struct {
	Amount  float64
	Optname string
}

// IPurchaseRequest represents a record in purchases table in the database.
type IPurchaseRequest interface {
	// Returns the db ID of the purchase request.
	Id() int64

	// Returns the unique UID of the purchase request.
	Uid() string

	// Returns the name of the purchase item.
	Name() string

	// Returns the price
	Price() float64

	// Returns true if the purchase request has a fixed price.
	IsFixedPrice() bool

	// Create a payment for the purchase.
	CreatePayment(tx *sql.Tx, ctx context.Context, params CreatePaymentParams) error

	// Pay using the customers wallet.
	// The amount will be debitted from the wallet once the purchase request has been confirmed.
	PayWithWallet(tx *sql.Tx, ctx context.Context, amount float64) error

	// Returns the state of the purchase.
	// The state includes the total accumulated payment for the purchase and other important details.
	State(tx *sql.Tx, ctx context.Context) (PurchaseState, error)

	// Executes the webhook for the purchase.
	// This will make an internal POST request to the webhook route.
	// The params contain the success status and message to be passed to the webhook handler.
	Execute(ctx context.Context, params ExecuteParams) error

	// Redirects the user to the callback route of the purchase request.
	RedirectToCallback(w http.ResponseWriter, r *http.Request)

	// Confirm the purchase.
	// This must be executed in the purchase callback handler.
	Confirm(tx *sql.Tx, ctx context.Context) error

	// Cancel the purchase.
	// This must be executed in the purchase callback handler.
	Cancel(tx *sql.Tx, ctx context.Context) error
}
