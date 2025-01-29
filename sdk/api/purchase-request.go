/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

// PurchaseRequest represents a purchase to be made by the customer.
type PurchaseRequest struct {
	Sku                 string
	Name                string
	Description         string
	Price               float64
	AnyPrice            bool
	CallbackRoute       string
	CallbackRouteParams map[string]string
}
