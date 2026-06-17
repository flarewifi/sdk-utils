/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

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
