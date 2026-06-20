/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"context"
	"net/http"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// Re-export currency types from sdk-utils
type SupportedCurrency = sdkutils.SupportedCurrency

const (
	CurrencyPhilippinePeso = sdkutils.CurrencyPhilippinePeso
	CurrencyUsDollar       = sdkutils.CurrencyUsDollar
	CurrencyNigerianNaira  = sdkutils.CurrencyNigerianNaira
)

type PaymentOption struct {
	UUID        string // Unique, stable identifier (16-char hash based on device property like MAC address)
	Name        string // Display label for the user
	RouteName   string
	RouteParams map[string]string
}

// IPaymentProvider represents a payment provider.
// A payment provider can have many payment options.
type IPaymentProvider interface {
	// Returns name of the payment provider.
	Name() string

	// Returns a list of available payment options.
	OptionsFactory(r *http.Request) []PaymentOption
}

// IPaymentsApi is used to handle customer payments.
type IPaymentsApi interface {

	// Registers a new payment provider.
	// The provider's payment options will become available for the customers.
	NewPaymentProvider(IPaymentProvider)

	// Creates a purchase request and prompts the user for payment.
	// It sends HTTP response and must be put as last line in the handler function.
	Checkout(w http.ResponseWriter, r *http.Request, p PurchaseRequest)

	// Returns the pending purchase for the client device.
	GetPurchaseRequest(r *http.Request) (IPurchaseRequest, error)

	// Returns the purchase request by its UUID.
	FindPurchaseRequestByUUID(uuid string) (IPurchaseRequest, error)

	// Formats a float64 amount as currency string using the current application currency.
	FormatCurrency(amount float64) string

	// ExtractPurchaseData extracts and validates purchase data from the request.
	// The purchase token is passed as a query parameter "token" (?token=<jwt>).
	// It verifies the JWT token signed with application secret,
	// extracts the purchase UUID and device ID from claims, and returns the purchase request.
	// This method handles both callback requests (GET) and webhook requests (POST).
	// Token expires after 5 minutes for security.
	// Returns the purchase request if successful, or an error if validation fails.
	ExtractPurchaseData(r *http.Request) (IPurchaseRequest, error)

	// CreatePurchase creates a purchase record programmatically without HTTP checkout flow.
	// Used for admin-generated purchases like voucher batch sales where no customer device is involved.
	// DeviceID can be nil for admin purchases.
	CreatePurchase(ctx context.Context, params CreatePurchaseParams) (IPurchaseRequest, error)

	// HandlePurchaseExecute registers an in-process handler invoked when a
	// payment provider calls IPurchaseRequest.Execute() for a purchase whose
	// WebHookRoute matches `route` and whose callback plugin is this plugin.
	// It replaces the former POST webhook route: register the handler instead
	// of mounting an HTTP endpoint. The `route` string is used purely as the
	// dispatch key and no longer needs to be a registered HTTP route.
	HandlePurchaseExecute(route string, handler PurchaseExecuteHandler)
}
