/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import "net/http"

// IInAppPurchasesApi is used to perform purchases and subscriptions.
type IInAppPurchasesApi interface {
	// Verify if user has already purchased the item.
	CheckOneTimePurchase(productID string) (InAppOneTimePurchaseStatus, error)

	// Verify if user has already subscribed to the item.
	CheckSubscription(planID string) (InAppSubscriptionStatus, error)

	// This will redirect the user to the purchase page to perform the transaction.
	PurchaseGuardMiddleware(InAppOneTimePurchaseStatus) (middleware func(next http.Handler) http.Handler)

	// This will redirect the user to the subscription page to perform the transaction.
	SubscriptionGuardMiddleware(InAppSubscription) (middleware func(next http.Handler) http.Handler)
}

// InAppOneTimePurchaseStatus represents an item that can be purchased.
type InAppPurchaseStatus string

var (
	InAppPurchaseStatusUnpaid InAppPurchaseStatus = "unpaid"
	InAppPurchaseStatusPaid   InAppPurchaseStatus = "paid"
)

type InAppOneTimePurchaseStatus struct {
	// The product ID of the item. You can get the product ID from your developer console dashboard.
	ProductID string
	Status    InAppPurchaseStatus
	Message   string
}

type InAppOneTimePurchase struct {
	ProductID       string
	DisplayPrice    float64
	DisplayCurrency string
}

type InAppSubscriptionFrequency string

var (
	InAppSubscriptionMonthly InAppSubscriptionFrequency = "monthly"
	InAppSubscriptionYearly  InAppSubscriptionFrequency = "yearly"
)

// InAppSubscription represents an in-app subscription item.
type InAppSubscriptionStatus struct {
	// PlanId is the ID of the subscription plan.
	PlanId  string
	Status  InAppPurchaseStatus
	Message string
}

type InAppSubscription struct {
	PlanId                string
	SubscriptionFrequency InAppSubscriptionFrequency
	DisplayPrice          float64 // display price based on customer/developer currency
	DisplayCurrency       float64 // display currency
}
