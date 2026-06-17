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

type IVoucherBatch interface {
	ID() int64
	UUID() string
	Amount() *float64
	Metadata() string                                                  // JSON metadata string
	ProviderPkg() string                                               // Plugin package that created this batch
	Vouchers(page int, perPage int, search string) ([]IVoucher, error) // Search filters vouchers by code, provider package or device mac
	VouchersCount() int64
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

// IVoucher represents a single voucher record.
type IVoucher interface {
	ID() int64
	UUID() string
	BatchUUID() string // returns the batch UUID that groups vouchers created together
	Code() string
	ProviderPkg() string // plugin that generated the voucher
	Type() SessionType
	TimeSecs() int64
	DataMb() int64
	DownSpeedMbps() int64
	UpSpeedMbps() int64
	SessionExpDays() *int    // nil means session never expires
	UseGlobal() bool         // true = use global bandwidth, false = use per-user bandwidth
	Session() IClientSession // returns associated session (only for activated vouchers)
	Device() IClientDevice   // returns associated device (only for activated vouchers)
	ExpiresAt() *time.Time   // nil means voucher never expires
	ActivatedAt() *time.Time // nil if not yet activated
	CreatedAt() time.Time
}

// CreateVouchersParams holds parameters for creating a batch of vouchers.
type CreateVouchersParams struct {
	Count          int
	Type           SessionType
	TimeSecs       int64
	DataMb         int64
	DownSpeedMbps  int64      // default 10 Mbps if 0
	UpSpeedMbps    int64      // default 10 Mbps if 0
	SessionExpDays *int       // nil means session never expires
	UseGlobal      bool       // true = use global bandwidth, false = use per-user bandwidth (default: false)
	ExpiresAt      *time.Time // nil means voucher never expires
	BatchUUID      string     // optional - if empty, a UUID will be generated
	Amount         *float64   // optional amount for the voucher batch
}

// UpdateVoucherParams holds parameters for updating a voucher.
type UpdateVoucherParams struct {
	ID             int64
	Type           SessionType
	TimeSecs       int64
	DataMb         int64
	DownSpeedMbps  int64
	UpSpeedMbps    int64
	SessionExpDays *int       // nil means session never expires
	UseGlobal      bool       // true = use global bandwidth, false = use per-user bandwidth
	ExpiresAt      *time.Time // nil means voucher never expires
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
	DateStart   *time.Time
	DateEnd     *time.Time
}

// UpdateVoucherBatchParams holds parameters for updating a voucher batch.
type UpdateVoucherBatchParams struct {
	UUID     string   // batch UUID to update
	Amount   *float64 // new amount (nil to clear)
	Metadata string   // new metadata (empty string to clear)
}

// ListVoucherBatchesParams holds parameters for listing voucher batches with pagination.
type ListVoucherBatchesParams struct {
	Search  *string
	Page    int
	PerPage int
}

// ListVoucherBatchesResult holds the result of listing voucher batches.
type ListVoucherBatchesResult struct {
	Batches []IVoucherBatch
	Count   int64
}

type ListVouchersResult struct {
	Vouchers []IVoucher
	Count    int64
}

// IVouchersApi manages voucher lifecycle including creation, activation, and deletion.
// Each plugin gets its own scoped instance — vouchers are filtered by the plugin's package name.
type IVouchersApi interface {
	// CreateVouchers creates a batch of vouchers and returns them.
	// Emits EventVoucherGenerated with the created voucher batch.
	CreateVouchers(ctx context.Context, params CreateVouchersParams) ([]IVoucher, error)

	// FindByCode finds an available (unactivated) voucher by its code.
	FindByCode(ctx context.Context, code string) (IVoucher, error)

	// FindByID finds a voucher by its database ID.
	FindByID(ctx context.Context, id int64) (IVoucher, error)

	// List returns a paginated list of vouchers for this plugin.
	List(ctx context.Context, params ListVouchersParams) (ListVouchersResult, error)

	// CountVouchers returns the total count of vouchers matching the filter criteria.
	CountVouchers(ctx context.Context, params ListVouchersParams) (int64, error)

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

	// GetVouchersByBatchUUIDCount returns the count of vouchers with the given batch UUID.
	GetVouchersByBatchUUIDCount(ctx context.Context, batchUUID string) (int64, error)

	// FindBatchByUUID finds a voucher batch by its UUID.
	FindBatchByUUID(ctx context.Context, batchUUID string) (IVoucherBatch, error)

	// FindBatchByCode finds a batch that contains a voucher with this code
	FindBatchByCode(ctx context.Context, code string) (IVoucherBatch, error)

	// UpdateBatch updates a voucher batch's amount and metadata.
	UpdateBatch(ctx context.Context, params UpdateVoucherBatchParams) (IVoucherBatch, error)

	// ListBatches returns a paginated list of voucher batches.
	ListBatches(ctx context.Context, params ListVoucherBatchesParams) (ListVoucherBatchesResult, error)

	// CountBatches returns the total count of voucher batches matching the filter criteria.
	CountBatches(ctx context.Context, params ListVoucherBatchesParams) (int64, error)

	// DeleteBatch removes a voucher batch and all its vouchers by UUID.
	// Emits EventVoucherBatchDeleted with the deleted batch.
	DeleteBatch(ctx context.Context, batchUUID string) error

	// OnVoucherEvent registers a callback to be called when a voucher event occurs.
	//
	// Deprecated: Use api.Events().OnVoucherEvent(...) instead.
	OnVoucherEvent(event VoucherEvent, callback func(ctx context.Context, v IVoucher) error)

	// OnVoucherBatchEvent registers a callback to be called when vouchers are generated.
	//
	// Deprecated: Use api.Events().OnVoucherBatchEvent(...) instead.
	OnVoucherBatchEvent(event VoucherEvent, callback func(ctx context.Context, batch IVoucherBatch) error)

}
