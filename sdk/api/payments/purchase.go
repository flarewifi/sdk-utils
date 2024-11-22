/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkpayments

import "net/http"

// IPurchase represents a record in purchases table in the database.
type IPurchase interface {

    // Returns the name of the purchase item.
	Name() string

    // Returns true if the purchase request has a fixed price.
	FixedPrice() (float64, bool)

    // Create a payment for the purchase.
	CreatePayment(amount float64, optname string) error

    // Pay using the customers wallet.
    // The amount will be debitted from the wallet once the purchase request has been confirmed.
	PayWithWallet(amount float64) error

    // Returns the state of the purchase.
    // The state includes the total accumulated payment for the purchase and other important details.
	State() (PurchaseState, error)

    // Executes the payment for the purchase.
    // This will redirect the user to the callback URL of purchase request.
	Execute(w http.ResponseWriter)

    // Confirm the purchase.
    // This must be executed in the purchase callback handler.
	Confirm() error

    // Cancel the purchase.
    // This must be executed in the purchase callback handler.
	Cancel() error
}
