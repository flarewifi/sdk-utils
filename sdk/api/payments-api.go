/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"net/http"
)

// List of supported currencies
const (
	CurrencyPhilippinePeso string = "PHP"
	CurrencyUsDollar       string = "USD"
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

	// Returns the purchase request by its unique identifier.
	GetPurchaseRequestByUID(uid string) (IPurchaseRequest, error)
}
