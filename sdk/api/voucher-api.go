/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

import (
	"context"
	"time"
)

// IVoucher represents a single voucher record.
type IVoucher interface {
	ID() int64
	UUID() string
	Code() string
	ProviderPkg() string // plugin that generated the voucher
	Type() SessionType
	TimeSecs() int64
	DataMb() int64
	DownSpeedMbps() int64
	UpSpeedMbps() int64
	SessionExpDays() *int         // nil means session never expires
	UseGlobal() bool              // true = use global bandwidth, false = use per-user bandwidth
	Session() IClientSession      // returns associated session (only for activated vouchers)
	Device() IClientDevice        // returns associated device (only for activated vouchers)
	VoucherExpiresOn() *time.Time // nil means voucher never expires (renamed from ExpiresOn)
	ActivatedAt() *time.Time      // nil if not yet activated
	CreatedAt() time.Time
	BatchUUID() string // returns the batch UUID that groups vouchers created together
}

// CreateVouchersParams holds parameters for creating a batch of vouchers.
type CreateVouchersParams struct {
	Count            int
	Type             SessionType
	TimeSecs         int64
	DataMb           int64
	DownSpeedMbps    int64      // default 10 Mbps if 0
	UpSpeedMbps      int64      // default 10 Mbps if 0
	SessionExpDays   *int       // nil means session never expires
	UseGlobal        bool       // true = use global bandwidth, false = use per-user bandwidth (default: false)
	VoucherExpiresOn *time.Time // nil means voucher never expires (renamed from ExpiresOn)
	TotalAmount      *float64   // optional amount to associate with each voucher (e.g. for paid vouchers)
	PaymentNote      *string    // optional note to associate with each voucher's payment (e.g. for paid vouchers)
}

// VoucherBatch represents a batch of vouchers with payment metadata
type VoucherBatch struct {
	ID          int64
	UUID        string
	TotalAmount *float64
	PaymentNote *string
	CreatedAt   time.Time
}

type VoucherBatchEvent struct {
	Vouchers    []IVoucher
	TotalAmount *float64
	PaymentNote *string
}

// UpdateVoucherParams holds parameters for updating a voucher.
type UpdateVoucherParams struct {
	ID               int64
	Type             SessionType
	TimeSecs         int64
	DataMb           int64
	DownSpeedMbps    int64
	UpSpeedMbps      int64
	SessionExpDays   *int       // nil means session never expires
	UseGlobal        bool       // true = use global bandwidth, false = use per-user bandwidth
	VoucherExpiresOn *time.Time // nil means voucher never expires (renamed from ExpiresOn)
}

// ActivateVoucherParams holds parameters for activating a voucher.
// Session is created internally based on voucher settings.
type ActivateVoucherParams struct {
	ID     int64         // Voucher ID
	Device IClientDevice // Device to associate with the voucher and session
}

// VoucherActivateResult holds the result of activating a voucher.
type VoucherActivateResult struct {
	Voucher IVoucher       // The activated voucher
	Session IClientSession // The session created from the voucher
}

// ListVouchersParams holds parameters for listing vouchers with pagination.
type ListVouchersParams struct {
	Search      *string // optional search term to filter vouchers by code, provider package or device mac
	IsActivated *bool   // optional filter to show only activated or unactivated vouchers
	Page        int
	PerPage     int
}

type ListVouchersResult struct {
	Vouchers []IVoucher
	Count    int64
}

// IVouchersApi manages voucher lifecycle including creation, activation, and deletion.
// Each plugin gets its own scoped instance — vouchers are filtered by the plugin's package name.
type IVouchersApi interface {
	// Create generates a batch of vouchers and returns them.
	// Emits EventVoucherGenerated with the created vouchers.
	Create(ctx context.Context, params CreateVouchersParams) ([]IVoucher, error)

	// FindByCode finds an available (unactivated) voucher by its code.
	FindByCode(ctx context.Context, code string) (IVoucher, error)

	// FindByID finds a voucher by its database ID.
	FindByID(ctx context.Context, id int64) (IVoucher, error)

	// List returns a paginated list of vouchers for this plugin.
	List(ctx context.Context, params ListVouchersParams) (ListVouchersResult, error)

	// Update changes a voucher's session type, time, data, and speed settings.
	// Emits EventVoucherUpdated with the updated voucher.
	Update(ctx context.Context, params UpdateVoucherParams) (IVoucher, error)

	// Activate marks a voucher as used, creates a session based on voucher settings,
	// and associates it with the provided device.
	// Emits EventVoucherActivated with the voucher.
	// Returns VoucherActivateResult containing the activated voucher and created session.
	Activate(ctx context.Context, params ActivateVoucherParams) (VoucherActivateResult, error)

	// Delete removes a voucher by its ID.
	// Emits EventVoucherDeleted with the deleted voucher.
	Delete(ctx context.Context, id int64) error

	// DeleteActivated removes all activated vouchers for this plugin.
	// Emits EventVoucherDeleted for each deleted voucher.
	DeleteActivated(ctx context.Context) error

	// GetAvailable returns all unactivated vouchers for this plugin.
	GetAvailable(ctx context.Context) ([]IVoucher, error)

	// FindVoucherBatch retrieves batch metadata by UUID.
	// Returns nil if batch not found.
	FindVoucherBatch(ctx context.Context, uuid string) (*VoucherBatch, error)

	// OnVoucherEvent registers a callback to be called when a voucher event occurs.
	OnVoucherEvent(event VoucherEvent, callback func(IVoucher) error)

	// OnVoucherBatchEvent registers a callback to be called when vouchers are generated.
	OnVoucherBatchEvent(event VoucherEvent, callback func([]IVoucher) error)

	// OnBeforeCreate registers a hook called before voucher creation.
	// The hook receives a pointer to params and can modify them.
	// Return an error to block creation.
	OnBeforeCreate(callback func(ctx context.Context, params *CreateVouchersParams) error)
}
