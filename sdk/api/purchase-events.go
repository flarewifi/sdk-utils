/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

// PurchaseEvent represents the type of a purchase event.
type PurchaseEvent string

const (
	// EventPurchaseSuccess is emitted when a purchase is successfully confirmed.
	EventPurchaseSuccess PurchaseEvent = "purchase:success"

	// EventPurchaseFailed is emitted when a purchase confirmation or execution fails.
	EventPurchaseFailed PurchaseEvent = "purchase:failed"

	// EventPurchaseCancelled is emitted when a purchase is cancelled by the user.
	EventPurchaseCancelled PurchaseEvent = "purchase:cancelled"
)

// PurchaseEventData represents the data associated with a purchase event.
type PurchaseEventData struct {
	// Purchase is the purchase request that triggered the event.
	Purchase IPurchaseRequest

	// Device is the client device associated with the purchase.
	Device IClientDevice

	// Reason provides context for failure or cancellation events.
	// Empty for success events.
	Reason string
}
