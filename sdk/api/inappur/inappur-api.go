/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkinappur

import "net/http"

// IInAppPurchasesApi is used to perform purchases and subscriptions.
type IInAppPurchasesApi interface {
	// Verify if user has already purchased the item.
	VerifyPurchase(InAppCheckoutItem) error

	// Verify if user has already subscribed to the item.
	VerifySubscription(InAppSubscriptionItem) error

	// This will redirect the user to the purchase page to perform the transaction.
	PurchaseGuardMiddleware(InAppCheckoutItem) (middleware func(next http.Handler) http.Handler)

	// This will redirect the user to the subscription page to perform the transaction.
	SubscriptionGuardMiddleware(InAppSubscriptionItem) (middleware func(next http.Handler) http.Handler)
}
