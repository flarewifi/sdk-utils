/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

// VoucherEvent represents the type of a voucher lifecycle event.
type VoucherEvent string

const (
	// EventVoucherGenerated is emitted after a batch of vouchers is created.
	EventVoucherGenerated VoucherEvent = "voucher:generated"

	// EventVoucherActivated is emitted when a voucher is used to start a session.
	EventVoucherActivated VoucherEvent = "voucher:activated"

	// EventVoucherUpdated is emitted when a voucher's validity is updated.
	EventVoucherUpdated VoucherEvent = "voucher:updated"

	// EventVoucherDeleted is emitted when a voucher is deleted.
	EventVoucherDeleted VoucherEvent = "voucher:deleted"

	// EventVoucherBeforeCreate is called before vouchers are created.
	// Hooks can modify params or return an error to block creation.
	EventVoucherBeforeCreate VoucherEvent = "voucher:before_create"
)
