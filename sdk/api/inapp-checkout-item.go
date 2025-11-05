/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

// InAppCheckoutItem represents an item that can be purchased.
type InAppCheckoutItem struct {
	// The product ID of the item. You can get the product ID from your developer console dashboard.
	ProductId string
}
