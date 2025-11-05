/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

// UnitAmount represents the payment amount and currency of items.
type UnitAmount struct {
	// Currency of the amount. Examples: USD, PHP, CNY.
	CurrencyCode string `query:"code"`

	// Numerical value of the amount.
	Value float64 `query:"val"`
}
