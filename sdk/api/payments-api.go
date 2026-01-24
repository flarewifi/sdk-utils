/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
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

	// FindPurchasesByPaymentOptionUUID returns all purchases made using the specified payment option UUID.
	// The confirmedOnly parameter determines whether to return all purchases or only confirmed ones.
	// Returns an empty slice if no purchases are found.
	FindPurchasesByPaymentOptionUUID(paymentOptionUUID string, confirmedOnly bool) ([]IPurchaseRequest, error)
}
